package langserver

import (
	"context"

	"github.com/radeksimko/terraform-ls/internal/filesystem"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
	fs := ctx.Value(ctxFs).(filesystem.Filesystem)
	return fs.Open(params.TextDocument)
}
