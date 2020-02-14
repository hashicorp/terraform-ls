package langserver

import (
	"context"

	lsp "github.com/sourcegraph/go-lsp"
)

func Exit(ctx context.Context, vs lsp.None) error {
	return nil
}
