package langserver

import (
	"context"

	lsp "github.com/sourcegraph/go-lsp"
)

func CancelRequest(ctx context.Context, vs lsp.CancelParams) error {
	return nil
}
