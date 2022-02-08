package handlers

import (
	"context"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) Initialized(ctx context.Context, params lsp.InitializedParams) error {
	// TODO: Initiate progress reporting from here?

	// Walker runs asynchronously so we're intentionally *not*
	// passing the request context here
	walkerCtx := context.Background()
	err := svc.walker.StartWalking(walkerCtx)
	if err != nil {
		return err
	}

	err = svc.watcher.Start(ctx)
	if err != nil {
		return err
	}

	return nil
}
