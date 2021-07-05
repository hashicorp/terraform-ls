package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) GoToReferenceTarget(ctx context.Context, params lsp.TextDocumentPositionParams) (interface{}, error) {
	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return nil, err
	}

	mf, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return nil, err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}

	mod, err := mf.ModuleByPath(file.Dir())
	if err != nil {
		return nil, err
	}

	schema, err := schemaForDocument(mf, file)
	if err != nil {
		return nil, err
	}

	d, err := decoderForDocument(ctx, mod, file.LanguageID())
	if err != nil {
		return nil, err
	}
	d.SetSchema(schema)

	fPos, err := ilsp.FilePositionFromDocumentPosition(params, file)
	if err != nil {
		return nil, err
	}

	h.logger.Printf("Looking for ref origin at %q -> %#v", file.Filename(), fPos.Position())
	origin, err := d.ReferenceOriginAtPos(file.Filename(), fPos.Position())
	if err != nil {
		return nil, err
	}
	if origin == nil {
		return nil, nil
	}

	target, err := d.ReferenceTargetForOrigin(*origin)
	if err != nil {
		return nil, err
	}

	return ilsp.ReferenceToLocationLink(mod.Path, *origin, target, cc.TextDocument.Declaration.LinkSupport), nil
}
