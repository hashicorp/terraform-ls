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
	"path"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
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
	"github.com/hashicorp/terraform-ls/internal/uri"
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

// GetTerraformVersion obtains "installed" Terraform version
// which can inform what version of core schema to pick.
// Knowing the version is not required though as we can rely on
// the constraint in `required_version` (as parsed via
// [LoadModuleMetadata] and compare it against known released versions.
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

// ObtainSchema obtains provider schemas via Terraform CLI.
// This is useful if we do not have the schemas available
// from the embedded FS (i.e. in [PreloadEmbeddedSchema]).
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

// PreloadEmbeddedSchema loads provider schemas based on
// provider requirements parsed earlier via [LoadModuleMetadata].
// This is the cheapest way of getting provider schemas in terms
// of resources, time and complexity/UX.
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

	for _, pAddr := range missingReqs {
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

// ParseModuleConfiguration parses the module configuration,
// i.e. turns bytes of `*.tf` files into AST ([*hcl.File]).
func ParseModuleConfiguration(ctx context.Context, fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known
	if mod.ModuleDiagnosticsState[ast.HCLParsingSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	var files ast.ModFiles
	var diags ast.ModDiags
	rpcContext := lsctx.DocumentContext(ctx)
	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if mod.ModuleDiagnosticsState[ast.HCLParsingSource] == op.OpStateLoaded && rpcContext.IsDidChangeRequest() && rpcContext.LanguageID == ilsp.Terraform.String() {
		// the file has already been parsed, so only examine this file and not the whole module
		err = modStore.SetModuleDiagnosticsState(modPath, ast.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		filePath, err := uri.PathFromURI(rpcContext.URI)
		if err != nil {
			return err
		}
		fileName := filepath.Base(filePath)

		f, fDiags, err := parser.ParseModuleFile(fs, filePath)
		if err != nil {
			return err
		}
		existingFiles := mod.ParsedModuleFiles.Copy()
		existingFiles[ast.ModFilename(fileName)] = f
		files = existingFiles

		existingDiags, ok := mod.ModuleDiagnostics[ast.HCLParsingSource]
		if !ok {
			existingDiags = make(ast.ModDiags)
		} else {
			existingDiags = existingDiags.Copy()
		}
		existingDiags[ast.ModFilename(fileName)] = fDiags
		diags = existingDiags
	} else {
		// this is the first time file is opened so parse the whole module
		err = modStore.SetModuleDiagnosticsState(modPath, ast.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		files, diags, err = parser.ParseModuleFiles(fs, modPath)
	}

	if err != nil {
		return err
	}

	sErr := modStore.UpdateParsedModuleFiles(modPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = modStore.UpdateModuleDiagnostics(modPath, ast.HCLParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}

// ParseVariables parses the variables configuration,
// i.e. turns bytes of `*.tfvars` files into AST ([*hcl.File]).
func ParseVariables(ctx context.Context, fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known
	if mod.VarsDiagnosticsState[ast.HCLParsingSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	var files ast.VarsFiles
	var diags ast.VarsDiags
	rpcContext := lsctx.RPCContext(ctx)
	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if mod.ModuleDiagnosticsState[ast.HCLParsingSource] == op.OpStateLoaded && rpcContext.IsDidChangeRequest() && lsctx.IsLanguageId(ctx, ilsp.Tfvars.String()) {
		// the file has already been parsed, so only examine this file and not the whole module
		err = modStore.SetVarsDiagnosticsState(modPath, ast.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}
		filePath, err := uri.PathFromURI(rpcContext.URI)
		if err != nil {
			return err
		}
		fileName := filepath.Base(filePath)

		f, vdiags, err := parser.ParseVariableFile(fs, filePath)
		if err != nil {
			return err
		}

		existingFiles := mod.ParsedVarsFiles.Copy()
		existingFiles[ast.VarsFilename(fileName)] = f
		files = existingFiles

		existingDiags, ok := mod.VarsDiagnostics[ast.HCLParsingSource]
		if !ok {
			existingDiags = make(ast.VarsDiags)
		} else {
			existingDiags = existingDiags.Copy()
		}
		existingDiags[ast.VarsFilename(fileName)] = vdiags
		diags = existingDiags
	} else {
		// this is the first time file is opened so parse the whole module
		err = modStore.SetVarsDiagnosticsState(modPath, ast.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		files, diags, err = parser.ParseVariableFiles(fs, modPath)
	}

	if err != nil {
		return err
	}

	sErr := modStore.UpdateParsedVarsFiles(modPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = modStore.UpdateVarsDiagnostics(modPath, ast.HCLParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}

// ParseModuleManifest parses the "module manifest" which
// contains records of installed modules, e.g. where they're
// installed on the filesystem.
// This is useful for processing any modules which are not local
// nor hosted in the Registry (which would be handled by
// [GetModuleDataFromRegistry]).
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

// ParseProviderVersions is a job complimentary to [ObtainSchema]
// in that it obtains versions of providers/schemas from Terraform
// CLI's lock file.
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

// LoadModuleMetadata loads data about the module in a version-independent
// way that enables us to decode the rest of the configuration,
// e.g. by knowing provider versions, Terraform Core constraint etc.
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

// DecodeReferenceTargets collects reference targets,
// using previously parsed AST (via [ParseModuleConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that variable block between certain LOC
// can be referred to as var.foobar. This is useful e.g. during completion,
// go-to-definition or go-to-references.
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

// DecodeReferenceOrigins collects reference origins,
// using previously parsed AST (via [ParseModuleConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
//
// For example it tells us that there is a reference address var.foobar
// at a particular LOC. This can be later matched with targets
// (as obtained via [DecodeReferenceTargets]) during hover or go-to-definition.
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

// DecodeVarsReferences collects reference origins within
// variable files (*.tfvars) where each valid attribute
// (as informed by schema provided via [LoadModuleMetadata])
// is considered an origin.
//
// This is useful in hovering over those variable names,
// go-to-definition and go-to-references.
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

// SchemaModuleValidation does schema-based validation
// of module files (*.tf) and produces diagnostics
// associated with any "invalid" parts of code.
//
// It relies on previously parsed AST (via [ParseModuleConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
func SchemaModuleValidation(ctx context.Context, modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.ModuleDiagnosticsState[ast.SchemaValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetModuleDiagnosticsState(modPath, ast.SchemaValidationSource, op.OpStateLoading)
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

	var rErr error
	rpcContext := lsctx.DocumentContext(ctx)
	if rpcContext.Method == "textDocument/didChange" && rpcContext.LanguageID == ilsp.Terraform.String() {
		filename := path.Base(rpcContext.URI)
		// We only revalidate a single file that changed
		var fileDiags hcl.Diagnostics
		fileDiags, rErr = moduleDecoder.ValidateFile(ctx, filename)

		modDiags, ok := mod.ModuleDiagnostics[ast.SchemaValidationSource]
		if !ok {
			modDiags = make(ast.ModDiags)
		}
		modDiags[ast.ModFilename(filename)] = fileDiags

		sErr := modStore.UpdateModuleDiagnostics(modPath, ast.SchemaValidationSource, modDiags)
		if sErr != nil {
			return sErr
		}
	} else {
		// We validate the whole module, e.g. on open
		var diags lang.DiagnosticsMap
		diags, rErr = moduleDecoder.Validate(ctx)

		sErr := modStore.UpdateModuleDiagnostics(modPath, ast.SchemaValidationSource, ast.ModDiagsFromMap(diags))
		if sErr != nil {
			return sErr
		}
	}

	return rErr
}

// SchemaVariablesValidation does schema-based validation
// of variable files (*.tfvars) and produces diagnostics
// associated with any "invalid" parts of code.
//
// It relies on previously parsed AST (via [ParseVariables])
// and schema, as provided via [LoadModuleMetadata]).
func SchemaVariablesValidation(ctx context.Context, modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.VarsDiagnosticsState[ast.SchemaValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetVarsDiagnosticsState(modPath, ast.SchemaValidationSource, op.OpStateLoading)
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
		LanguageID: ilsp.Tfvars.String(),
	})
	if err != nil {
		return err
	}

	var rErr error
	rpcContext := lsctx.DocumentContext(ctx)
	if rpcContext.Method == "textDocument/didChange" && rpcContext.LanguageID == ilsp.Tfvars.String() {
		filename := path.Base(rpcContext.URI)
		// We only revalidate a single file that changed
		var fileDiags hcl.Diagnostics
		fileDiags, rErr = moduleDecoder.ValidateFile(ctx, filename)

		varsDiags, ok := mod.VarsDiagnostics[ast.SchemaValidationSource]
		if !ok {
			varsDiags = make(ast.VarsDiags)
		}
		varsDiags[ast.VarsFilename(filename)] = fileDiags

		sErr := modStore.UpdateVarsDiagnostics(modPath, ast.SchemaValidationSource, varsDiags)
		if sErr != nil {
			return sErr
		}
	} else {
		// We validate the whole module, e.g. on open
		var diags lang.DiagnosticsMap
		diags, rErr = moduleDecoder.Validate(ctx)

		sErr := modStore.UpdateVarsDiagnostics(modPath, ast.SchemaValidationSource, ast.VarsDiagsFromMap(diags))
		if sErr != nil {
			return sErr
		}
	}

	return rErr
}

// ReferenceValidation does validation based on (mis)matched
// reference origins and targets, to flag up "orphaned" references.
//
// It relies on [DecodeReferenceTargets] and [DecodeReferenceOrigins]
// to supply both origins and targets to compare.
func ReferenceValidation(ctx context.Context, modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.ModuleDiagnosticsState[ast.ReferenceValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetModuleDiagnosticsState(modPath, ast.ReferenceValidationSource, op.OpStateLoading)
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

	diags := validations.UnreferencedOrigins(ctx, pathCtx)
	return modStore.UpdateModuleDiagnostics(modPath, ast.ReferenceValidationSource, ast.ModDiagsFromMap(diags))
}

// TerraformValidate uses Terraform CLI to run validate subcommand
// and turn the provided (JSON) output into diagnostics associated
// with "invalid" parts of code.
func TerraformValidate(ctx context.Context, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.ModuleDiagnosticsState[ast.TerraformValidateSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetModuleDiagnosticsState(modPath, ast.TerraformValidateSource, op.OpStateLoading)
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

// GetModuleDataFromRegistry obtains data about any modules (inputs & outputs)
// from the Registry API based on module calls which were previously parsed
// via [LoadModuleMetadata]. The same data could be obtained via [ParseModuleManifest]
// but getting it from the API comes with little expectations,
// specifically the modules do not need to be installed on disk and we don't
// need to parse and decode all files.
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
