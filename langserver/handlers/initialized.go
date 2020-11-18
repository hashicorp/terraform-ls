package handlers

import (
	"context"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func Initialized(ctx context.Context, params lsp.None) error {
	return nil
}
