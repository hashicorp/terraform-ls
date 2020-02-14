package langserver

import (
	"context"

	"github.com/radeksimko/terraform-ls/internal/filesystem"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentDidClose(ctx context.Context, params lsp.DidCloseTextDocumentParams) error {
	fs := ctx.Value(ctxFs).(filesystem.Filesystem)
	return fs.Close(params.TextDocument)
}
