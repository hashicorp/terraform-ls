package langserver

import (
	"context"

	lsp "github.com/sourcegraph/go-lsp"
)

func Initialize(ctx context.Context, params lsp.InitializeParams) (lsp.InitializeResult, error) {
	return lsp.InitializeResult{
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
	}, nil
}
