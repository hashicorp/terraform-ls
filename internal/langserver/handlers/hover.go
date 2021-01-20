package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
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

	schema, err := mf.SchemaForModule(file.Dir())
	if err != nil {
		return nil, err
	}

	d, err := module.DecoderForModule(mod)
	if err != nil {
		return nil, err
	}
	d.SetSchema(schema)

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
