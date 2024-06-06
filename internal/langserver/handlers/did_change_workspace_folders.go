// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/document"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (svc *service) DidChangeWorkspaceFolders(ctx context.Context, params lsp.DidChangeWorkspaceFoldersParams) error {
	for _, removed := range params.Event.Removed {
		if !uri.IsURIValid(removed.URI) {
			jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
				Type: lsp.Warning,
				Message: fmt.Sprintf("Ignoring workspace folder (unsupport or invalid URI) %s."+
					" This is most likely bug, please report it.", removed.URI),
			})
			continue
		}
		svc.removeIndexedModule(ctx, removed.URI)
	}

	for _, added := range params.Event.Added {
		if !uri.IsURIValid(added.URI) {
			jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
				Type: lsp.Warning,
				Message: fmt.Sprintf("Ignoring workspace folder (unsupport or invalid URI) %s."+
					" This is most likely bug, please report it.", added.URI),
			})
			continue
		}
		svc.indexNewModule(ctx, added.URI)
	}

	return nil
}

func (svc *service) indexNewModule(ctx context.Context, modURI string) {
	modHandle := document.DirHandleFromURI(modURI)

	err := svc.stateStore.WalkerPaths.EnqueueDir(ctx, modHandle)
	if err != nil {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("Ignoring new folder %s: %s."+
				" This is most likely bug, please report it.", modURI, err),
		})
		return
	}
}

func (svc *service) removeIndexedModule(ctx context.Context, modURI string) {
	modHandle := document.DirHandleFromURI(modURI)

	err := svc.stateStore.WalkerPaths.DequeueDir(modHandle)
	if err != nil {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("Ignoring removed folder %s: %s."+
				" This is most likely bug, please report it.", modURI, err),
		})
		return
	}

	err = svc.stateStore.JobStore.DequeueJobsForDir(modHandle)
	if err != nil {
		svc.logger.Printf("failed to dequeue jobs for module: %s", err)
		return
	}

	// callers, err := svc.stateStore.Roots.CallersOfModule(modHandle.Path())
	// if err != nil {
	// 	svc.logger.Printf("failed to remove module from watcher: %s", err)
	// 	return
	// }

	// if len(callers) == 0 {
	// 	err = svc.stateStore.Roots.Remove(modHandle.Path())
	// 	svc.logger.Printf("failed to remove records: %s", err)
	// }
}
