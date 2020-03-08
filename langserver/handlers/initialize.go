package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	lsp "github.com/sourcegraph/go-lsp"
)

func Initialize(ctx context.Context, params lsp.InitializeParams) (lsp.InitializeResult, error) {
	serverCaps := lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
				Options: &lsp.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    lsp.TDSKFull,
				},
			},
			CompletionProvider: &lsp.CompletionOptions{
				ResolveProvider: false,
			},
		},
	}

	err := lsctx.SetClientCapabilities(ctx, &params.Capabilities)

	return serverCaps, err
}
