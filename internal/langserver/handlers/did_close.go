package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func TextDocumentDidClose(ctx context.Context, params lsp.DidCloseTextDocumentParams) error {
	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return err
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)
	err = fs.CloseAndRemoveDocument(fh)
	if err != nil {
		return err
	}

	return nil
}
