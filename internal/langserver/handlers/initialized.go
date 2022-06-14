package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/go-uuid"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
)

func (svc *service) Initialized(ctx context.Context, params lsp.InitializedParams) error {
	caps, err := ilsp.ClientCapabilities(ctx)
	if err != nil {
		return err
	}

	return svc.setupWatchedFiles(ctx, caps.Workspace.DidChangeWatchedFiles)
}

func (svc *service) setupWatchedFiles(ctx context.Context, caps lsp.DidChangeWatchedFilesClientCapabilities) error {
	if !caps.DynamicRegistration {
		svc.logger.Printf("Client doesn't support dynamic watched files registration, " +
			"provider and module changes may not be reflected at runtime")
		return nil
	}

	id, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}

	watchPatterns := datadir.PathGlobPatternsForWatching()
	watchers := make([]lsp.FileSystemWatcher, len(watchPatterns))
	for i, wp := range watchPatterns {
		watchers[i] = lsp.FileSystemWatcher{
			GlobPattern: wp.Pattern,
			Kind:        kindFromEventType(wp.EventType),
		}
	}

	srv := jrpc2.ServerFromContext(ctx)
	_, err = srv.Callback(ctx, "client/registerCapability", lsp.RegistrationParams{
		Registrations: []lsp.Registration{
			{
				ID:     id,
				Method: "workspace/didChangeWatchedFiles",
				RegisterOptions: lsp.DidChangeWatchedFilesRegistrationOptions{
					Watchers: watchers,
				},
			},
		},
	})
	if err != nil {
		svc.logger.Printf("failed to register watched files: %s", err)
	}
	return nil
}

func kindFromEventType(eventType datadir.EventType) uint32 {
	switch eventType {
	case datadir.CreateEventType:
		return uint32(lsp.Created)
	case datadir.ModifyEventType:
		return uint32(lsp.Changed)
	case datadir.DeleteEventType:
		return uint32(lsp.Deleted)
	}
	return 0
}
