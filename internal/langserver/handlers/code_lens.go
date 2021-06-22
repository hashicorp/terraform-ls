package handlers

import (
	"context"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) TextDocumentCodeLens(ctx context.Context, params lsp.CodeLensParams) ([]lsp.CodeLens, error) {
	// TODO: Implement code lens
	return []lsp.CodeLens{}, nil
}
