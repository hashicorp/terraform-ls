// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/terraform-ls/internal/document"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (svc *service) TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
	docURI := string(params.TextDocument.URI)

	// URIs are always checked during initialize request, but
	// we still allow single-file mode, therefore invalid URIs
	// can still land here, so we check for those.
	if !uri.IsURIValid(docURI) {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("Ignoring workspace folder (unsupport or invalid URI) %s."+
				" This is most likely bug, please report it.", docURI),
		})
		return fmt.Errorf("invalid URI: %s", docURI)
	}

	dh := document.HandleFromURI(docURI)

	err := svc.stateStore.DocumentStore.OpenDocument(dh, params.TextDocument.LanguageID,
		int(params.TextDocument.Version), []byte(params.TextDocument.Text))
	if err != nil {
		return err
	}

	mod, err := svc.modStore.ModuleByPath(dh.Dir.Path())
	if err != nil {
		if state.IsModuleNotFound(err) {
			err = svc.modStore.Add(dh.Dir.Path())
			if err != nil {
				return err
			}
			mod, err = svc.modStore.ModuleByPath(dh.Dir.Path())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	svc.logger.Printf("opened module: %s", mod.Path)

	// We reparse because the file being opened may not match
	// (originally parsed) content on the disk
	// TODO: Do this only if we can verify the file differs?
	modHandle := document.DirHandleFromPath(mod.Path)
	jobIds, err := svc.indexer.DocumentOpened(ctx, modHandle)
	if err != nil {
		return err
	}

	if svc.singleFileMode {
		err = svc.stateStore.WalkerPaths.EnqueueDir(ctx, modHandle)
		if err != nil {
			return err
		}
	}

	return svc.stateStore.JobStore.WaitForJobs(ctx, jobIds...)
}
