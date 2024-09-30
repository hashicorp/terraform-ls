// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"
	"os"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (svc *service) DidChangeWatchedFiles(ctx context.Context, params lsp.DidChangeWatchedFilesParams) error {
	svc.logger.Printf("Received changes %q", len(params.Changes))

	for _, change := range params.Changes {
		svc.logger.Printf("Received change event for %q: %s", change.Type, change.URI)
		rawURI := string(change.URI)

		// This is necessary because clients may not send delete notifications
		// for individual nested files when the parent directory is deleted.
		// VS Code / vscode-languageclient behaves this way.
		// If the .terraform directory changes
		if modUri, ok := datadir.ModuleUriFromDataDir(rawURI); ok {
			// If the .terraform directory is deleted,
			// we need to clear the module manifest
			if change.Type == lsp.Deleted {
				// This is unlikely to happen unless the user manually removed files
				// See https://github.com/hashicorp/terraform/issues/30005
				modHandle := document.DirHandleFromURI(modUri)
				svc.eventBus.ManifestChange(eventbus.ManifestChangeEvent{
					Context:    ctx, // We pass the context for data here
					Dir:        modHandle,
					ChangeType: lsp.Deleted,
				})
			}

			continue // Ignore any other changes to the .terraform directory
		}

		// If the .terraform.lock.hcl (or older implementation) file changes
		if modUri, ok := datadir.ModuleUriFromPluginLockFile(rawURI); ok {
			if change.Type == lsp.Deleted {
				// This is unlikely to happen unless the user manually removed files
				// See https://github.com/hashicorp/terraform/issues/30005
				// Cached provider schema could be removed here but it may be useful
				// in other modules, so we trade some memory for better UX here.
				continue
			}

			modHandle := document.DirHandleFromURI(modUri)
			svc.eventBus.PluginLockChange(eventbus.PluginLockChangeEvent{
				Context:    ctx, // We pass the context for data here
				Dir:        modHandle,
				ChangeType: change.Type,
			})

			continue
		}

		// If the .terraform/modules/modules.json file changes
		if modUri, ok := datadir.ModuleUriFromModuleLockFile(rawURI); ok {
			modHandle := document.DirHandleFromURI(modUri)
			svc.eventBus.ManifestChange(eventbus.ManifestChangeEvent{
				Context:    ctx, // We pass the context for data here
				Dir:        modHandle,
				ChangeType: change.Type,
			})

			continue
		}

		// If the .terraform/modules/terraform-sources.json file changes
		if modUri, ok := datadir.ModuleUriFromTerraformSourcesFile(rawURI); ok {
			modHandle := document.DirHandleFromURI(modUri)
			// manifest change event handles terraform-sources.json as well
			svc.eventBus.ManifestChange(eventbus.ManifestChangeEvent{
				Context:    ctx, // We pass the context for data here
				Dir:        modHandle,
				ChangeType: change.Type,
			})

			continue
		}

		rawPath, err := uri.PathFromURI(rawURI)
		if err != nil {
			svc.logger.Printf("error parsing %q: %s", rawURI, err)
			continue
		}
		isDir := false

		if change.Type == lsp.Deleted {
			// Fall through and just fire the event
		}

		if change.Type == lsp.Changed {
			// Check if document is open and skip running any jobs
			// as we already did so as part of textDocument/didChange
			// which clients should always send for *open* documents
			// even if they change outside of the IDE.
			docHandle := document.HandleFromURI(rawURI)
			isOpen, err := svc.stateStore.DocumentStore.IsDocumentOpen(docHandle)
			if err != nil {
				svc.logger.Printf("error when checking open document (%q changed): %s", rawURI, err)
			}
			if isOpen {
				svc.logger.Printf("document is open - ignoring event for %q", rawURI)
				continue
			}

			fi, err := os.Stat(rawPath)
			if err != nil {
				svc.logger.Printf("error checking existence (%q changed): %s", rawPath, err)
				continue
			}
			if fi.IsDir() {
				isDir = true
			}
		}

		if change.Type == lsp.Created {
			fi, err := os.Stat(rawPath)
			if err != nil {
				svc.logger.Printf("error checking existence (%q created): %s", rawPath, err)
				continue
			}

			// If the path is a directory, enqueue it for walking and wait for
			// it to be walked. This will ensure that the features received
			// discover events for all the new directories
			if fi.IsDir() {
				dir := document.DirHandleFromPath(rawPath)
				isDir = true
				err = svc.stateStore.WalkerPaths.EnqueueDir(ctx, dir)
				if err != nil {
					jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
						Type: lsp.Warning,
						Message: fmt.Sprintf("Failed to walk path %q: %s."+
							" This is most likely bug, please report it.", rawURI, err),
					})
				}
				err = svc.stateStore.WalkerPaths.WaitForDirs(ctx, []document.DirHandle{dir})
				if err != nil {
					jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
						Type: lsp.Warning,
						Message: fmt.Sprintf("Failed to wait for path walk %q: %s."+
							" This is most likely bug, please report it.", rawURI, err),
					})
				}
			}
		}

		svc.eventBus.DidChangeWatched(eventbus.DidChangeWatchedEvent{
			Context:    ctx, // We pass the context for data here
			RawPath:    rawPath,
			IsDir:      isDir,
			ChangeType: change.Type,
		})
	}

	return nil
}
