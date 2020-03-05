package langserver

import (
	"context"

	lsp "github.com/sourcegraph/go-lsp"
)

func Initialized(ctx context.Context, params lsp.None) error {
	return nil
}
