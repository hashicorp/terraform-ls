package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) DidChangeWorkspaceFolders(ctx context.Context, params lsp.DidChangeWorkspaceFoldersParams) error {
	watcher, err := lsctx.Watcher(ctx)
	if err != nil {
		return err
	}

	for _, removed := range params.Event.Removed {
		modHandle := document.DirHandleFromURI(removed.URI)

		err := svc.stateStore.WalkerPaths.DequeueDir(modHandle)
		if err != nil {
			jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
				Type: lsp.Warning,
				Message: fmt.Sprintf("Ignoring removed workspace folder %s: %s."+
					" This is most likely bug, please report it.", removed.URI, err),
			})
			continue
		}

		err = watcher.RemoveModule(modHandle.Path())
		if err != nil {
			svc.logger.Printf("failed to remove module from watcher: %s", err)
			continue
		}

		err = svc.stateStore.JobStore.DequeueJobsForDir(modHandle)
		if err != nil {
			svc.logger.Printf("failed to dequeue jobs for module: %s", err)
			continue
		}

		callers, err := svc.modStore.CallersOfModule(modHandle.Path())
		if err != nil {
			svc.logger.Printf("failed to remove module from watcher: %s", err)
			continue
		}
		if len(callers) == 0 {
			err = svc.modStore.Remove(modHandle.Path())
			svc.logger.Printf("failed to remove module: %s", err)
		}
	}

	for _, added := range params.Event.Added {
		modHandle := document.DirHandleFromURI(added.URI)

		err = svc.stateStore.WalkerPaths.EnqueueDir(modHandle)
		if err != nil {
			jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
				Type: lsp.Warning,
				Message: fmt.Sprintf("Ignoring new workspace folder %s: %s."+
					" This is most likely bug, please report it.", added.URI, err),
			})
			continue
		}

		err = watcher.AddModule(modHandle.Path())
		if err != nil {
			svc.logger.Printf("failed to add module to watcher: %s", err)
			continue
		}
	}

	return nil
}
