// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package module

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	tfjson "github.com/hashicorp/terraform-json"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/decoder/validations"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/terraform/parser"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/earlydecoder"
	tfmodule "github.com/hashicorp/terraform-schema/module"
	tfregistry "github.com/hashicorp/terraform-schema/registry"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/hashicorp/terraform-ls/internal/terraform/module"

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

	// Avoid getting version if getting is already in progress or already known
	if mod.TerraformVersionState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetTerraformVersionState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}
	defer modStore.SetTerraformVersionState(modPath, op.OpStateLoaded)

	tfExec, err := TerraformExecutorForModule(ctx, mod.Path)
	if err != nil {
		sErr := modStore.UpdateTerraformAndProviderVersions(modPath, nil, nil, err)
		if err != nil {
			return sErr
		}
		return err
	}

	v, pv, err := tfExec.Version(ctx)

	// TODO: Remove and rely purely on ParseProviderVersions
	// In most cases we get the provider version from the datadir/lockfile
	// but there is an edge case with custom plugin location
	// when this may not be available, so leveraging versions
	// from "terraform version" accounts for this.
	// See https://github.com/hashicorp/terraform-ls/issues/24
	pVersions := providerVersionsFromTfVersion(pv)

	sErr := modStore.UpdateTerraformAndProviderVersions(modPath, v, pVersions, err)
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

	// Avoid obtaining schema if it is already in progress or already known
	if mod.ProviderSchemaState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	pReqs, err := modStore.ProviderRequirementsForModule(modPath)
	if err != nil {
		return err
	}

	exist, err := schemaStore.AllSchemasExist(pReqs)
	if err != nil {
		return err
	}
	if exist {
		// avoid obtaining schemas if we already have it
		return nil
	}

	tfExec, err := TerraformExecutorForModule(ctx, modPath)
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

func PreloadEmbeddedSchema(ctx context.Context, logger *log.Logger, fs fs.ReadDirFS, modStore *state.ModuleStore, schemaStore *state.ProviderSchemaStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid preloading schema if it is already in progress or already known
	if mod.PreloadEmbeddedSchemaState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetPreloadEmbeddedSchemaState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}
	defer modStore.SetPreloadEmbeddedSchemaState(modPath, op.OpStateLoaded)

	pReqs, err := modStore.ProviderRequirementsForModule(modPath)
	if err != nil {
		return err
	}

	missingReqs, err := schemaStore.MissingSchemas(pReqs)
	if err != nil {
		return err
	}
	if len(missingReqs) == 0 {
		// avoid preloading any schemas if we already have all
		return nil
	}

	for pAddr := range missingReqs {
		err := preloadSchemaForProviderAddr(ctx, pAddr, fs, schemaStore, logger)
		if err != nil {
			return err
		}
	}

	return nil
}

func preloadSchemaForProviderAddr(ctx context.Context, pAddr tfaddr.Provider, fs fs.ReadDirFS,
	schemaStore *state.ProviderSchemaStore, logger *log.Logger) error {

	startTime := time.Now()

	if pAddr.IsLegacy() && pAddr.Type == "terraform" {
		// The terraform provider is built into Terraform 0.11+
		// and while it's possible, users typically don't declare
		// entry in required_providers block for it.
		pAddr = tfaddr.NewProvider(tfaddr.BuiltInProviderHost, tfaddr.BuiltInProviderNamespace, "terraform")
	} else if pAddr.IsLegacy() {
		// Since we use recent version of Terraform to generate
		// embedded schemas, these will never contain legacy
		// addresses.
		//
		// A legacy namespace may come from missing
		// required_providers entry & implied requirement
		// from the provider block or 0.12-style entry,
		// such as { grafana = "1.0" }.
		//
		// Implying "hashicorp" namespace here mimics behaviour
		// of all recent (0.14+) Terraform versions.
		originalAddr := pAddr
		pAddr.Namespace = "hashicorp"
		logger.Printf("preloading schema for %s (implying %s)",
			originalAddr.ForDisplay(), pAddr.ForDisplay())
	}

	ctx, rootSpan := otel.Tracer(tracerName).Start(ctx, "preloadProviderSchema",
		trace.WithAttributes(attribute.KeyValue{
			Key:   attribute.Key("ProviderAddress"),
			Value: attribute.StringValue(pAddr.String()),
		}))
	defer rootSpan.End()

	pSchemaFile, err := schemas.FindProviderSchemaFile(fs, pAddr)
	if err != nil {
		rootSpan.RecordError(err)
		rootSpan.SetStatus(codes.Error, "schema file not found")
		if errors.Is(err, schemas.SchemaNotAvailable{Addr: pAddr}) {
			logger.Printf("preloaded schema not available for %s", pAddr)
			return nil
		}
		return err
	}

	_, span := otel.Tracer(tracerName).Start(ctx, "readProviderSchemaFile",
		trace.WithAttributes(attribute.KeyValue{
			Key:   attribute.Key("ProviderAddress"),
			Value: attribute.StringValue(pAddr.String()),
		}))
	b, err := io.ReadAll(pSchemaFile.File)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "schema file not readable")
		return err
	}
	span.SetStatus(codes.Ok, "schema file read successfully")
	span.End()

	_, span = otel.Tracer(tracerName).Start(ctx, "decodeProviderSchemaData",
		trace.WithAttributes(attribute.KeyValue{
			Key:   attribute.Key("ProviderAddress"),
			Value: attribute.StringValue(pAddr.String()),
		}))
	jsonSchemas := tfjson.ProviderSchemas{}
	err = json.Unmarshal(b, &jsonSchemas)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "schema file not decodable")
		return err
	}
	span.SetStatus(codes.Ok, "schema data decoded successfully")
	span.End()

	ps, ok := jsonSchemas.Schemas[pAddr.String()]
	if !ok {
		return fmt.Errorf("%q: no schema found in file", pAddr)
	}

	pSchema := tfschema.ProviderSchemaFromJson(ps, pAddr)
	pSchema.SetProviderVersion(pAddr, pSchemaFile.Version)

	_, span = otel.Tracer(tracerName).Start(ctx, "loadProviderSchemaDataIntoMemDb",
		trace.WithAttributes(attribute.KeyValue{
			Key:   attribute.Key("ProviderAddress"),
			Value: attribute.StringValue(pAddr.String()),
		}))
	err = schemaStore.AddPreloadedSchema(pAddr, pSchemaFile.Version, pSchema)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "loading schema into mem-db failed")
		span.End()
		existsError := &state.AlreadyExistsError{}
		if errors.As(err, &existsError) {
			// This accounts for a possible race condition
			// where we may be preloading the same schema
			// for different providers at the same time
			logger.Printf("schema for %s is already loaded", pAddr)
			return nil
		}
		return err
	}
	span.SetStatus(codes.Ok, "schema loaded successfully")
	span.End()

	elapsedTime := time.Now().Sub(startTime)
	logger.Printf("preloaded schema for %s %s in %s", pAddr, pSchemaFile.Version, elapsedTime)
	rootSpan.SetStatus(codes.Ok, "schema loaded successfully")

	return nil
}

func ParseModuleConfiguration(ctx context.Context, fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// TODO: Only parse the file that's being changed/opened, unless this is 1st-time parsing

	// Avoid parsing if it is already in progress or already known
	if mod.ModuleParsingState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetModuleParsingState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	files, diags, err := parser.ParseModuleFiles(fs, modPath)

	sErr := modStore.UpdateParsedModuleFiles(modPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = modStore.UpdateModuleDiagnostics(modPath, ast.ModuleParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}

func ParseVariables(ctx context.Context, fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// TODO: Only parse the file that's being changed/opened, unless this is 1st-time parsing

	// Avoid parsing if it is already in progress or already known
	if mod.VarsParsingState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetVarsParsingState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	files, diags, err := parser.ParseVariableFiles(fs, modPath)

	sErr := modStore.UpdateParsedVarsFiles(modPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = modStore.UpdateVarsDiagnostics(modPath, ast.VarsParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}

func ParseModuleManifest(ctx context.Context, fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid parsing if it is already in progress or already known
	if mod.ModManifestState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetModManifestState(modPath, op.OpStateLoading)
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

func ParseProviderVersions(ctx context.Context, fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid parsing if it is already in progress or already known
	if mod.InstalledProvidersState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetInstalledProvidersState(modPath, op.OpStateLoading)
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

func LoadModuleMetadata(ctx context.Context, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if upstream (parsing) job reported no changes

	// Avoid parsing if it is already in progress or already known
	if mod.MetaState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetMetaState(modPath, op.OpStateLoading)
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

	providerRefs := make(map[tfmodule.ProviderRef]tfaddr.Provider, len(meta.ProviderReferences))
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
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.RefTargetsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetReferenceTargetsState(modPath, op.OpStateLoading)
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
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.RefOriginsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetReferenceOriginsState(modPath, op.OpStateLoading)
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
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream (parsing) job reported no changes

	// Avoid collection if it is already in progress or already done
	if mod.VarsRefOriginsState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetVarsReferenceOriginsState(modPath, op.OpStateLoading)
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

func EarlyValidation(ctx context.Context, modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.DiagnosticsState[ast.SchemaValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetDiagnosticsState(modPath, ast.SchemaValidationSource, op.OpStateLoading)
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

	diags, rErr := moduleDecoder.Validate(ctx)
	sErr := modStore.UpdateModuleDiagnostics(modPath, ast.SchemaValidationSource, ast.ModDiagsFromMap(diags))
	if sErr != nil {
		return sErr
	}

	return rErr
}

func ReferenceValidation(ctx context.Context, modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.DiagnosticsState[ast.ReferenceValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetDiagnosticsState(modPath, ast.ReferenceValidationSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	pathReader := &idecoder.PathReader{
		ModuleReader: modStore,
		SchemaReader: schemaReader,
	}
	pathCtx, err := pathReader.PathContext(lang.Path{
		Path:       modPath,
		LanguageID: ilsp.Terraform.String(),
	})
	if err != nil {
		return err
	}

	ctx = decoder.WithPathContext(ctx, pathCtx)

	diags := validations.UnreferencedOrigins(ctx)
	return modStore.UpdateModuleDiagnostics(modPath, ast.ReferenceValidationSource, ast.ModDiagsFromMap(diags))
}

func TerraformValidate(ctx context.Context, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.DiagnosticsState[ast.TerraformValidateSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetDiagnosticsState(modPath, ast.TerraformValidateSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	tfExec, err := TerraformExecutorForModule(ctx, mod.Path)
	if err != nil {
		return err
	}

	jsonDiags, err := tfExec.Validate(ctx)
	if err != nil {
		return err
	}
	validateDiags := diagnostics.HCLDiagsFromJSON(jsonDiags)

	return modStore.UpdateModuleDiagnostics(modPath, ast.TerraformValidateSource, ast.ModDiagsFromMap(validateDiags))
}

func GetModuleDataFromRegistry(ctx context.Context, regClient registry.Client, modStore *state.ModuleStore, modRegStore *state.RegistryModuleStore, modPath string) error {
	// loop over module calls
	calls, err := modStore.ModuleCalls(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid collection if upstream jobs (parsing, meta) reported no changes

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

			clientError := registry.ClientError{}
			if errors.As(err, &clientError) &&
				((clientError.StatusCode >= 400 && clientError.StatusCode < 408) ||
					(clientError.StatusCode > 408 && clientError.StatusCode < 429)) {
				// Still cache the module
				err = modRegStore.CacheError(sourceAddr)
				if err != nil {
					errs = multierror.Append(errs, err)
				}
			}

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
				typ, err := ctyjson.UnmarshalType(rawType)
				if err == nil {
					inputType = typ
				}
			}
			inputs[i].Type = inputType

			if input.Default != "" {
				// Registry API unfortunately doesn't marshal values using
				// cty marshalers, making it lossy, so we just try to decode
				// on best-effort basis.
				val, err := ctyjson.Unmarshal([]byte(input.Default), inputType)
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
