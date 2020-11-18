package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) TextDocumentHover(ctx context.Context, params lsp.TextDocumentPositionParams) (lsp.Hover, error) {
	var data lsp.Hover

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return data, err
	}

	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return data, err
	}

	rmf, err := lsctx.RootModuleFinder(ctx)
	if err != nil {
		return data, err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return data, err
	}

	rm, err := rmf.RootModuleByPath(file.Dir())
	if err != nil {
		return data, err
	}

	schema, err := rmf.SchemaForPath(file.Dir())
	if err != nil {
		return data, err
	}

	d, err := rm.DecoderWithSchema(schema)
	if err != nil {
		return data, err
	}

	fPos, err := ilsp.FilePositionFromDocumentPosition(params, file)
	if err != nil {
		return data, err
	}

	h.logger.Printf("Looking for hover data at %q -> %#v", file.Filename(), fPos.Position())
	hoverData, err := d.HoverAtPos(file.Filename(), fPos.Position())
	h.logger.Printf("received hover data: %#v", data)
	if err != nil {
		return data, err
	}

	return ilsp.HoverData(hoverData, cc.TextDocument), nil
}
