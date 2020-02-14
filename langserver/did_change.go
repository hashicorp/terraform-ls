package langserver

import (
	"context"

	"github.com/radeksimko/terraform-ls/internal/filesystem"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentDidChange(ctx context.Context, params lsp.DidChangeTextDocumentParams) error {
	fs := ctx.Value(ctxFs).(filesystem.Filesystem)
	return fs.Change(params.TextDocument, params.ContentChanges)
}
