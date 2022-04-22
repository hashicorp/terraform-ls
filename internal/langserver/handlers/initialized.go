package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func Initialized(ctx context.Context, params lsp.InitializedParams) error {

	// dynamic registraion

	jrpc2.ServerFromContext(ctx).Callback(ctx, "client/registerCapability", lsp.RegistrationParams{
		Registrations: []lsp.Registration{
			{
				ID:     "79eee87c-c409-4664-8102-e03263673f6f",
				Method: "workspace/didChangeWatchedFiles",
				RegisterOptions: lsp.DidChangeWatchedFilesRegistrationOptions{
					Watchers: []lsp.FileSystemWatcher{
						{
							GlobPattern: "**/*.tf",
						},
						{
							GlobPattern: "**/*.tfvars",
						},
					},
				},
			},
		},
	})

	return nil
}
