package langserver

import (
	"context"

	lsctx "github.com/radeksimko/terraform-ls/internal/context"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return err
	}

	return fs.Open(params.TextDocument)
}
