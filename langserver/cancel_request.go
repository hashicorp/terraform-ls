package langserver

import (
	"context"

	"github.com/creachadair/jrpc2"
	lsp "github.com/sourcegraph/go-lsp"
)

func CancelRequest(ctx context.Context, params lsp.CancelParams) error {
	id := params.ID.String()
	jrpc2.CancelRequest(ctx, id)
	return nil
}
