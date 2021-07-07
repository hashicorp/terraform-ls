package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) References(ctx context.Context, params lsp.ReferenceParams) ([]lsp.Location, error) {
	list := make([]lsp.Location, 0)

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return list, err
	}

	mf, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return list, err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return list, err
	}

	mod, err := mf.ModuleByPath(file.Dir())
	if err != nil {
		return list, err
	}

	schema, err := schemaForDocument(mf, file)
	if err != nil {
		return list, err
	}

	d, err := decoderForDocument(ctx, mod, file.LanguageID())
	if err != nil {
		return list, err
	}
	d.SetSchema(schema)

	fPos, err := ilsp.FilePositionFromDocumentPosition(params.TextDocumentPositionParams, file)
	if err != nil {
		return list, err
	}

	refTarget, err := d.InnermostReferenceTargetAtPos(fPos.Filename(), fPos.Position())
	if err != nil {
		return list, err
	}
	if refTarget == nil {
		// this position is not addressable
		h.logger.Printf("position is not addressable: %s - %#v", fPos.Filename(), fPos.Position())
		return list, nil
	}

	h.logger.Printf("finding origins for inner-most target: %#v", refTarget)

	origins, err := d.ReferenceOriginsTargeting(*refTarget)
	if err != nil {
		return list, err
	}

	return ilsp.RefOriginsToLocations(mod.Path, origins), nil
}
