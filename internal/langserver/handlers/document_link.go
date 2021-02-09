package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/decoder"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) TextDocumentLink(ctx context.Context, params lsp.DocumentLinkParams) ([]lsp.DocumentLink, error) {
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

	d, err := decoder.DecoderForModule(ctx, mod)
	if err != nil {
		return nil, err
	}
	d.SetSchema(schema)

	links, err := d.LinksInFile(file.Filename())
	if err != nil {
		return nil, err
	}

	return ilsp.Links(links, cc.TextDocument.DocumentLink), nil
}
