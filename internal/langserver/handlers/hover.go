package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) TextDocumentHover(ctx context.Context, params lsp.TextDocumentPositionParams) (*lsp.Hover, error) {
	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return nil, err
	}

	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	rmf, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return nil, err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}

	rm, err := rmf.ModuleByPath(file.Dir())
	if err != nil {
		return nil, err
	}

	schema, err := rmf.SchemaForPath(file.Dir())
	if err != nil {
		return nil, err
	}

	d, err := rm.DecoderWithSchema(schema)
	if err != nil {
		return nil, err
	}

	fPos, err := ilsp.FilePositionFromDocumentPosition(params, file)
	if err != nil {
		return nil, err
	}

	h.logger.Printf("Looking for hover data at %q -> %#v", file.Filename(), fPos.Position())
	hoverData, err := d.HoverAtPos(file.Filename(), fPos.Position())
	h.logger.Printf("received hover data: %#v", hoverData)
	if err != nil {
		return nil, err
	}

	return ilsp.HoverData(hoverData, cc.TextDocument), nil
}
