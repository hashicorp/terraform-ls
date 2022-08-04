package module

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/terraform/parser"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/earlydecoder"
	"github.com/hashicorp/terraform-schema/module"
	tfregistry "github.com/hashicorp/terraform-schema/registry"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/json"
)

type DeferFunc func(opError error)

type ModuleOperation struct {
	ModulePath string
	Type       op.OpType
	Defer      DeferFunc

	doneCh chan struct{}
}

func NewModuleOperation(modPath string, typ op.OpType) ModuleOperation {
	return ModuleOperation{
		ModulePath: modPath,
		Type:       typ,
		doneCh:     make(chan struct{}, 1),
	}
}

func (mo ModuleOperation) markAsDone() {
	mo.doneCh <- struct{}{}
	close(mo.doneCh)
}

func (mo ModuleOperation) done() <-chan struct{} {
	return mo.doneCh
}

func GetTerraformVersion(ctx context.Context, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	err = modStore.SetTerraformVersionState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}
	defer modStore.SetTerraformVersionState(modPath, op.OpStateLoaded)

	tfExec, err := TerraformExecutorForModule(ctx, mod.Path)
	if err != nil {
		sErr := modStore.UpdateTerraformVersion(modPath, nil, nil, err)
		if err != nil {
			return sErr
		}
		return err
	}

	v, pv, err := tfExec.Version(ctx)
	pVersions := providerVersionsFromTfVersion(pv)

	sErr := modStore.UpdateTerraformVersion(modPath, v, pVersions, err)
	if sErr != nil {
		return sErr
	}

	return err
}

func providerVersionsFromTfVersion(pv map[string]*version.Version) map[tfaddr.Provider]*version.Version {
	m := make(map[tfaddr.Provider]*version.Version, 0)

	for rawAddr, v := range pv {
		pAddr, err := tfaddr.ParseProviderSource(rawAddr)
		if err != nil {
			// skip unparsable address
			continue
		}
		if pAddr.IsLegacy() {
			// TODO: check for migrations via Registry API?
		}
		m[pAddr] = v
	}

	return m
}

func ObtainSchema(ctx context.Context, modStore *state.ModuleStore, schemaStore *state.ProviderSchemaStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	tfExec, err := TerraformExecutorForModule(ctx, mod.Path)
	if err != nil {
		sErr := modStore.FinishProviderSchemaLoading(modPath, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	ps, err := tfExec.ProviderSchemas(ctx)
	if err != nil {
		sErr := modStore.FinishProviderSchemaLoading(modPath, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	for rawAddr, pJsonSchema := range ps.Schemas {
		pAddr, err := tfaddr.ParseProviderSource(rawAddr)
		if err != nil {
			// skip unparsable address
			continue
		}

		if pAddr.IsLegacy() {
			// TODO: check for migrations via Registry API?
		}

		pSchema := tfschema.ProviderSchemaFromJson(pJsonSchema, pAddr)

		err = schemaStore.AddLocalSchema(modPath, pAddr, pSchema)
		if err != nil {
			return err
		}
	}

	return nil
}

func ParseModuleConfiguration(fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	err := modStore.SetModuleParsingState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	files, diags, err := parser.ParseModuleFiles(fs, modPath)

	sErr := modStore.UpdateParsedModuleFiles(modPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = modStore.UpdateModuleDiagnostics(modPath, diags)
	if sErr != nil {
		return sErr
	}

	return err
}

func ParseVariables(fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	err := modStore.SetVarsParsingState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	files, diags, err := parser.ParseVariableFiles(fs, modPath)

	sErr := modStore.UpdateParsedVarsFiles(modPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = modStore.UpdateVarsDiagnostics(modPath, diags)
	if sErr != nil {
		return sErr
	}

	return err
}

func ParseModuleManifest(fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	err := modStore.SetModManifestState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	manifestPath, ok := datadir.ModuleManifestFilePath(fs, modPath)
	if !ok {
		err := fmt.Errorf("%s: manifest file does not exist", modPath)
		sErr := modStore.UpdateModManifest(modPath, nil, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	mm, err := datadir.ParseModuleManifestFromFile(manifestPath)
	if err != nil {
		err := fmt.Errorf("failed to parse manifest: %w", err)
		sErr := modStore.UpdateModManifest(modPath, nil, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	sErr := modStore.UpdateModManifest(modPath, mm, err)

	if sErr != nil {
		return sErr
	}
	return err
}

func ParseProviderVersions(fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	err := modStore.SetInstalledProvidersState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	pvm, err := datadir.ParsePluginVersions(fs, modPath)

	sErr := modStore.UpdateInstalledProviders(modPath, pvm, err)
	if sErr != nil {
		return sErr
	}

	return err
}

func LoadModuleMetadata(modStore *state.ModuleStore, modPath string) error {
	err := modStore.SetMetaState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	var mErr error
	meta, diags := earlydecoder.LoadModule(mod.Path, mod.ParsedModuleFiles.AsMap())
	if len(diags) > 0 {
		mErr = diags
	}

	providerRequirements := make(map[tfaddr.Provider]version.Constraints, len(meta.ProviderRequirements))
	for pAddr, pvc := range meta.ProviderRequirements {
		// TODO: check pAddr for migrations via Registry API?
		providerRequirements[pAddr] = pvc
	}
	meta.ProviderRequirements = providerRequirements

	providerRefs := make(map[module.ProviderRef]tfaddr.Provider, len(meta.ProviderReferences))
	for localRef, pAddr := range meta.ProviderReferences {
		// TODO: check pAddr for migrations via Registry API?
		providerRefs[localRef] = pAddr
	}
	meta.ProviderReferences = providerRefs

	sErr := modStore.UpdateMetadata(modPath, meta, mErr)
	if sErr != nil {
		return sErr
	}
	return mErr
}

func DecodeReferenceTargets(ctx context.Context, modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	err := modStore.SetReferenceTargetsState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&idecoder.PathReader{
		ModuleReader: modStore,
		SchemaReader: schemaReader,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	pd, err := d.Path(lang.Path{
		Path:       modPath,
		LanguageID: ilsp.Terraform.String(),
	})
	if err != nil {
		return err
	}
	targets, rErr := pd.CollectReferenceTargets()

	targets = append(targets, builtinReferences(modPath)...)

	sErr := modStore.UpdateReferenceTargets(modPath, targets, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

func DecodeReferenceOrigins(ctx context.Context, modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	err := modStore.SetReferenceOriginsState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&idecoder.PathReader{
		ModuleReader: modStore,
		SchemaReader: schemaReader,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	moduleDecoder, err := d.Path(lang.Path{
		Path:       modPath,
		LanguageID: ilsp.Terraform.String(),
	})
	if err != nil {
		return err
	}

	origins, rErr := moduleDecoder.CollectReferenceOrigins()

	sErr := modStore.UpdateReferenceOrigins(modPath, origins, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

func DecodeVarsReferences(ctx context.Context, modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	err := modStore.SetVarsReferenceOriginsState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&idecoder.PathReader{
		ModuleReader: modStore,
		SchemaReader: schemaReader,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	varsDecoder, err := d.Path(lang.Path{
		Path:       modPath,
		LanguageID: ilsp.Tfvars.String(),
	})
	if err != nil {
		return err
	}

	origins, rErr := varsDecoder.CollectReferenceOrigins()
	sErr := modStore.UpdateVarsReferenceOrigins(modPath, origins, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

func GetModuleDataFromRegistry(ctx context.Context, regClient registry.Client, modStore *state.ModuleStore, modRegStore *state.RegistryModuleStore, modPath string) error {
	// loop over module calls
	calls, err := modStore.ModuleCalls(modPath)
	if err != nil {
		return err
	}

	var errs *multierror.Error

	for _, declaredModule := range calls.Declared {
		sourceAddr, ok := declaredModule.SourceAddr.(tfaddr.Module)
		if !ok {
			// skip any modules which do not come from the Registry
			continue
		}

		// check if that address was already cached
		// if there was an error finding in cache, so cache again
		exists, err := modRegStore.Exists(sourceAddr, declaredModule.Version)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		if exists {
			// entry in cache, no need to look up
			continue
		}

		// get module data from Terraform Registry
		metaData, err := regClient.GetModuleData(ctx, sourceAddr, declaredModule.Version)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		inputs := make([]tfregistry.Input, len(metaData.Root.Inputs))
		for i, input := range metaData.Root.Inputs {
			inputs[i] = tfregistry.Input{
				Name:        input.Name,
				Description: lang.Markdown(input.Description),
				Required:    input.Required,
			}

			inputType := cty.DynamicPseudoType
			if input.Type != "" {
				// Registry API unfortunately doesn't marshal types using
				// cty marshalers, making it lossy, so we just try to decode
				// on best-effort basis.
				rawType := []byte(fmt.Sprintf("%q", input.Type))
				typ, err := json.UnmarshalType(rawType)
				if err == nil {
					inputType = typ
				}
			}
			inputs[i].Type = inputType

			if input.Default != "" {
				// Registry API unfortunately doesn't marshal values using
				// cty marshalers, making it lossy, so we just try to decode
				// on best-effort basis.
				val, err := json.Unmarshal([]byte(input.Default), inputType)
				if err == nil {
					inputs[i].Default = val
				}
			}
		}
		outputs := make([]tfregistry.Output, len(metaData.Root.Outputs))
		for i, output := range metaData.Root.Outputs {
			outputs[i] = tfregistry.Output{
				Name:        output.Name,
				Description: lang.Markdown(output.Description),
			}
		}

		modVersion, err := version.NewVersion(metaData.Version)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		// if not, cache it
		err = modRegStore.Cache(sourceAddr, modVersion, inputs, outputs)
		if err != nil {
			// A different job which ran in parallel for a different module block
			// with the same source may have already cached the same module.
			existsError := &state.AlreadyExistsError{}
			if errors.As(err, &existsError) {
				continue
			}

			errs = multierror.Append(errs, err)
			continue
		}
	}

	return errs.ErrorOrNil()
}
