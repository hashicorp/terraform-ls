package module

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/earlydecoder"
	"github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
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
}

func (mo ModuleOperation) Done() <-chan struct{} {
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
	pVersions := providerVersions(pv)

	sErr := modStore.UpdateTerraformVersion(modPath, v, pVersions, err)
	if err != nil {
		return sErr
	}
	return err
}

func providerVersions(pv map[string]*version.Version) map[tfaddr.Provider]*version.Version {
	m := make(map[tfaddr.Provider]*version.Version, 0)

	for rawAddr, v := range pv {
		pAddr, err := tfaddr.ParseRawProviderSourceString(rawAddr)
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
		pAddr, err := tfaddr.ParseRawProviderSourceString(rawAddr)
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

func ParseModuleConfiguration(fs filesystem.Filesystem, modStore *state.ModuleStore, modPath string) error {
	err := modStore.SetModuleParsingState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	files := make(map[string]*hcl.File, 0)
	diags := make(map[string]hcl.Diagnostics, 0)

	infos, err := fs.ReadDir(modPath)
	if err != nil {
		sErr := modStore.UpdateParsedModuleFiles(modPath, files, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		if !strings.HasSuffix(name, ".tf") || IsIgnoredFile(name) {
			continue
		}
		// TODO: overrides

		fullPath := filepath.Join(modPath, name)

		src, err := fs.ReadFile(fullPath)
		if err != nil {
			sErr := modStore.UpdateParsedModuleFiles(modPath, files, err)
			if sErr != nil {
				return sErr
			}
			return err
		}

		f, pDiags := hclsyntax.ParseConfig(src, name, hcl.InitialPos)
		diags[name] = pDiags
		if f != nil {
			files[name] = f
		}
	}

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

func ParseVariables(fs filesystem.Filesystem, modStore *state.ModuleStore, modPath string) error {
	err := modStore.SetVarsParsingState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	files := make(map[string]*hcl.File, 0)
	diags := make(map[string]hcl.Diagnostics, 0)

	infos, err := fs.ReadDir(modPath)
	if err != nil {
		sErr := modStore.UpdateParsedVarsFiles(modPath, files, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		if !(strings.HasSuffix(name, ".auto.tfvars") ||
			name == "terraform.tfvars") || IsIgnoredFile(name) {
			continue
		}

		fullPath := filepath.Join(modPath, name)

		src, err := fs.ReadFile(fullPath)
		if err != nil {
			sErr := modStore.UpdateParsedVarsFiles(modPath, files, err)
			if sErr != nil {
				return sErr
			}
			return err
		}

		f, pDiags := hclsyntax.ParseConfig(src, name, hcl.InitialPos)
		diags[name] = pDiags
		if f != nil {
			files[name] = f
		}
	}

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

// IsIgnoredFile returns true if the given filename (which must not have a
// directory path ahead of it) should be ignored as e.g. an editor swap file.
func IsIgnoredFile(name string) bool {
	return strings.HasPrefix(name, ".") || // Unix-like hidden files
		strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#") // emacs
}

func ParseModuleManifest(fs filesystem.Filesystem, modStore *state.ModuleStore, modPath string) error {
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
	meta, diags := earlydecoder.LoadModule(mod.Path, mod.ParsedModuleFiles)
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

func DecodeReferenceTargets(modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	err := modStore.SetReferenceTargetsState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder()
	for name, f := range mod.ParsedModuleFiles {
		err := d.LoadFile(name, f)
		if err != nil {
			return fmt.Errorf("failed to load a file: %w", err)
		}
	}

	fullSchema, schemaErr := schemaForModule(mod, schemaReader)
	if schemaErr != nil {
		sErr := modStore.UpdateReferenceTargets(modPath, lang.ReferenceTargets{}, schemaErr)
		if sErr != nil {
			return sErr
		}
		return schemaErr
	}
	d.SetSchema(fullSchema)

	targets, rErr := d.CollectReferenceTargets()

	bRefs := builtinReferences(modPath)
	targets = append(targets, bRefs...)

	sErr := modStore.UpdateReferenceTargets(modPath, targets, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}

func DecodeReferenceOrigins(modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	err := modStore.SetReferenceOriginsState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder()
	for name, f := range mod.ParsedModuleFiles {
		err := d.LoadFile(name, f)
		if err != nil {
			return fmt.Errorf("failed to load a file: %w", err)
		}
	}

	fullSchema, schemaErr := schemaForModule(mod, schemaReader)
	if schemaErr != nil {
		sErr := modStore.UpdateReferenceOrigins(modPath, lang.ReferenceOrigins{}, schemaErr)
		if sErr != nil {
			return sErr
		}
		return schemaErr
	}
	d.SetSchema(fullSchema)

	origins, rErr := d.CollectReferenceOrigins()

	sErr := modStore.UpdateReferenceOrigins(modPath, origins, rErr)
	if sErr != nil {
		return sErr
	}

	return rErr
}
