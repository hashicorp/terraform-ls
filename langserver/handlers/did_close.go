package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentDidClose(ctx context.Context, params lsp.DidCloseTextDocumentParams) error {
	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return err
	}

	return fs.Close(params.TextDocument)
}
