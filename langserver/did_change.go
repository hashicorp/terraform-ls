package langserver

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentDidChange(ctx context.Context, params lsp.DidChangeTextDocumentParams) error {
	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return err
	}

	return fs.Change(params.TextDocument, params.ContentChanges)
}
