package decoder

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/codelens"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func NewDecoder(ctx context.Context, pathReader decoder.PathReader) *decoder.Decoder {
	d := decoder.NewDecoder(pathReader)
	d.SetContext(decoderContext(ctx))
	return d
}

func modulePathContext(mod *state.Module, schemaReader state.SchemaReader, modReader ModuleReader) (*decoder.PathContext, error) {
	schema, err := schemaForModule(mod, schemaReader, modReader)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
	}

	for _, origin := range mod.RefOrigins {
		if ast.IsModuleFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}
	for _, target := range mod.RefTargets {
		if target.RangePtr != nil && ast.IsModuleFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		} else if target.RangePtr == nil {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	for name, f := range mod.ParsedModuleFiles {
		pathCtx.Files[name.String()] = f
	}

	return pathCtx, nil
}

func varsPathContext(mod *state.Module) (*decoder.PathContext, error) {
	schema, err := tfschema.SchemaForVariables(mod.Meta.Variables, mod.Path)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
	}

	for _, origin := range mod.VarsRefOrigins {
		if ast.IsVarsFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}
	for _, target := range mod.RefTargets {
		if target.RangePtr != nil && ast.IsVarsFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	for name, f := range mod.ParsedVarsFiles {
		pathCtx.Files[name.String()] = f
	}
	return pathCtx, nil
}

func decoderContext(ctx context.Context) decoder.DecoderContext {
	dCtx := decoder.DecoderContext{
		UtmSource:     "terraform-ls",
		UseUtmContent: true,
	}

	cc, err := ilsp.ClientCapabilities(ctx)
	if err == nil {
		cmdId, ok := lsp.ExperimentalClientCapabilities(cc.Experimental).ShowReferencesCommandId()
		if ok {
			dCtx.CodeLenses = append(dCtx.CodeLenses, codelens.ReferenceCount(cmdId))
		}
	}

	clientName, ok := ilsp.ClientName(ctx)
	if ok {
		dCtx.UtmMedium = clientName
	}
	return dCtx
}
