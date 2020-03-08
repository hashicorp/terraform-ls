package handlers

import (
	"context"

	lsp "github.com/sourcegraph/go-lsp"
)

func Shutdown(ctx context.Context, vs lsp.None) error {
	return nil
}
