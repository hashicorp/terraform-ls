// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (svc *service) DidChangeWatchedFiles(ctx context.Context, params lsp.DidChangeWatchedFilesParams) error {
	var ids job.IDs

	for _, change := range params.Changes {
		rawURI := string(change.URI)

		// This is necessary because clients may not send delete notifications
		// for individual nested files when the parent directory is deleted.
		// VS Code / vscode-languageclient behaves this way.
		if modUri, ok := datadir.ModuleUriFromDataDir(rawURI); ok {
			modHandle := document.DirHandleFromURI(modUri)
			if change.Type == protocol.Deleted {
				// This is unlikely to happen unless the user manually removed files
				// See https://github.com/hashicorp/terraform/issues/30005
				err := svc.dirStores.Modules.UpdateModManifest(modHandle.Path(), nil, nil)
				if err != nil {
					svc.logger.Printf("failed to remove module manifest for %q: %s", modHandle, err)
				}
			}
			continue
		}

		if modUri, ok := datadir.ModuleUriFromPluginLockFile(rawURI); ok {
			if change.Type == protocol.Deleted {
				// This is unlikely to happen unless the user manually removed files
				// See https://github.com/hashicorp/terraform/issues/30005
				// Cached provider schema could be removed here but it may be useful
				// in other modules, so we trade some memory for better UX here.
				continue
			}

			modHandle := document.DirHandleFromURI(modUri)
			err := svc.indexDirIfNotExists(ctx, modHandle)
			if err != nil {
				svc.logger.Printf("failed to index module %q: %s", modHandle, err)
				continue
			}

			jobIds, err := svc.indexer.PluginLockChanged(ctx, modHandle)
			if err != nil {
				svc.logger.Printf("error refreshing plugins for %q: %s", rawURI, err)
				continue
			}
			ids = append(ids, jobIds...)
			continue
		}

		if modUri, ok := datadir.ModuleUriFromModuleLockFile(rawURI); ok {
			modHandle := document.DirHandleFromURI(modUri)
			if change.Type == protocol.Deleted {
				// This is unlikely to happen unless the user manually removed files
				// See https://github.com/hashicorp/terraform/issues/30005
				err := svc.dirStores.Modules.UpdateModManifest(modHandle.Path(), nil, nil)
				if err != nil {
					svc.logger.Printf("failed to remove module manifest for %q: %s", modHandle, err)
				}
				continue
			}

			err := svc.indexDirIfNotExists(ctx, modHandle)
			if err != nil {
				svc.logger.Printf("failed to index module %q: %s", modHandle, err)
				continue
			}

			jobIds, err := svc.indexer.ModuleManifestChanged(ctx, modHandle)
			if err != nil {
				svc.logger.Printf("error refreshing plugins for %q: %s", modHandle, err)
				continue
			}
			ids = append(ids, jobIds...)
			continue
		}

		rawPath, err := uri.PathFromURI(rawURI)
		if err != nil {
			svc.logger.Printf("error parsing %q: %s", rawURI, err)
			continue
		}

		if change.Type == protocol.Deleted {
			// We don't know whether file or dir is being deleted
			// 1st we just blindly try to look it up as a directory
			_, err = svc.dirStores.Modules.ModuleByPath(rawPath)
			if err == nil {
				svc.removeIndexedModule(ctx, rawURI)
				continue
			}

			// 2nd we try again assuming it is a file
			parentDir := filepath.Dir(rawPath)
			_, err = svc.dirStores.Modules.ModuleByPath(parentDir)
			if err != nil {
				svc.logger.Printf("error finding module (%q deleted): %s", parentDir, err)
				continue
			}

			// and check the parent directory still exists
			fi, err := os.Stat(parentDir)
			if err != nil {
				if os.IsNotExist(err) {
					// if not, we remove the indexed module
					svc.removeIndexedModule(ctx, rawURI)
					continue
				}
				svc.logger.Printf("error checking existence (%q deleted): %s", parentDir, err)
				continue
			}
			if !fi.IsDir() {
				svc.logger.Printf("error: %q (deleted) is not a directory", parentDir)
				continue
			}

			// if the parent directory exists, we just need to
			// reparse the module after a file was deleted from it
			dirHandle := document.DirHandleFromPath(parentDir)
			jobIds, err := svc.indexer.DocumentChanged(ctx, dirHandle)
			if err != nil {
				svc.logger.Printf("error parsing module (%q deleted): %s", rawURI, err)
				continue
			}

			ids = append(ids, jobIds...)
		}

		if change.Type == protocol.Changed {
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

			ph, err := handleFromRawOsPath(ctx, rawPath)
			if err != nil {
				if err == ErrorSkip {
					continue
				}
				svc.logger.Printf("error (%q changed): %s", rawURI, err)
				continue
			}

			_, err = svc.dirStores.Modules.ModuleByPath(ph.DirHandle.Path())
			if err != nil {
				svc.logger.Printf("error finding module (%q changed): %s", rawURI, err)
				continue
			}

			jobIds, err := svc.indexer.DocumentChanged(ctx, ph.DirHandle)
			if err != nil {
				svc.logger.Printf("error parsing module (%q changed): %s", rawURI, err)
				continue
			}

			ids = append(ids, jobIds...)
		}

		if change.Type == protocol.Created {
			ph, err := handleFromRawOsPath(ctx, rawPath)
			if err != nil {
				if err == ErrorSkip {
					continue
				}
				svc.logger.Printf("error (%q created): %s", rawURI, err)
				continue
			}

			if ph.IsDir {
				err = svc.stateStore.WalkerPaths.EnqueueDir(ctx, ph.DirHandle)
				if err != nil {
					jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
						Type: lsp.Warning,
						Message: fmt.Sprintf("Ignoring new folder %s: %s."+
							" This is most likely bug, please report it.", rawURI, err),
					})
					continue
				}
			} else {
				jobIds, err := svc.indexer.DocumentChanged(ctx, ph.DirHandle)
				if err != nil {
					svc.logger.Printf("error parsing module (%q created): %s", rawURI, err)
					continue
				}

				ids = append(ids, jobIds...)
			}
		}
	}

	err := svc.stateStore.JobStore.WaitForJobs(ctx, ids...)
	if err != nil {
		return err
	}

	return nil
}

func (svc *service) indexDirIfNotExists(ctx context.Context, handle document.DirHandle) error {
	_, err := svc.dirStores.ByPath(handle.Path(), "terraform")
	if err != nil {
		if state.IsModuleNotFound(err) {
			err = svc.stateStore.WalkerPaths.EnqueueDir(ctx, handle)
			if err != nil {
				return fmt.Errorf("failed to walk module %q: %w", handle, err)
			}
			err = svc.stateStore.WalkerPaths.WaitForDirs(ctx, []document.DirHandle{handle})
			if err != nil {
				return fmt.Errorf("failed to wait for module walk %q: %w", handle, err)
			}
		} else {
			return fmt.Errorf("failed to find module %q: %w", handle, err)
		}
	}
	return nil
}

func handleFromRawOsPath(ctx context.Context, rawPath string) (*parsedHandle, error) {
	fi, err := os.Stat(rawPath)
	if err != nil {
		return nil, err
	}

	// URI can either be a file or a directory based on the LSP spec.
	if fi.IsDir() {
		return &parsedHandle{
			DirHandle: document.DirHandleFromPath(rawPath),
			IsDir:     true,
		}, nil
	}

	if !ast.IsSupportedFilename(fi.Name()) {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("Unable to update %q: filetype not supported. "+
				"This is likely a bug which should be reported.", rawPath),
		})
		return nil, ErrorSkip
	}

	docHandle := document.HandleFromPath(rawPath)
	return &parsedHandle{
		DirHandle: docHandle.Dir,
		IsDir:     false,
	}, nil
}

type parsedHandle struct {
	DirHandle document.DirHandle
	IsDir     bool
}

var ErrorSkip = errSkip{}

type errSkip struct{}

func (es errSkip) Error() string {
	return "skip"
}
