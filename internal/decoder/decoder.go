package decoder

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/codelens"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/state"
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
		ReferenceOrigins: mod.RefOrigins,
		ReferenceTargets: mod.RefTargets,
		Files:            make(map[string]*hcl.File, 0),
	}

	for name, f := range mod.ParsedModuleFiles {
		pathCtx.Files[name.String()] = f
	}

	return pathCtx, nil
}

func varsPathContext(mod *state.Module) (*decoder.PathContext, error) {
	schema, err := tfschema.SchemaForVariables(mod.Meta.Variables)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: mod.RefOrigins,
		ReferenceTargets: mod.RefTargets,
		Files:            make(map[string]*hcl.File, 0),
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
