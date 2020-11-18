package handlers

import (
	"context"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func Shutdown(ctx context.Context, vs lsp.None) error {
	return nil
}
