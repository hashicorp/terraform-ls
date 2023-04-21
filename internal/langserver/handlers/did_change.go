// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"time"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

var (
	jobTimer                 *time.Timer
	startProcessingJobsAfter = 500 * time.Millisecond
)

func (svc *service) TextDocumentDidChange(ctx context.Context, params lsp.DidChangeTextDocumentParams) error {
	p := lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: params.TextDocument.URI,
			},
			Version: params.TextDocument.Version,
		},
		ContentChanges: params.ContentChanges,
	}

	dh := ilsp.HandleFromDocumentURI(p.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return err
	}

	newVersion := int(p.TextDocument.Version)

	// Versions don't have to be consecutive, but they must be increasing
	if newVersion <= doc.Version {
		svc.logger.Printf("Old document version (%d) received, current version is %d. "+
			"Ignoring this update for %s. This is likely a client bug, please report it.",
			newVersion, doc.Version, p.TextDocument.URI)
		return nil
	}

	changes := ilsp.DocumentChanges(params.ContentChanges)
	newText, err := document.ApplyChanges(doc.Text, changes)
	if err != nil {
		return err
	}
	err = svc.stateStore.DocumentStore.UpdateDocument(dh, newText, newVersion)
	if err != nil {
		return err
	}

	// check existence
	_, err = svc.modStore.ModuleByPath(dh.Dir.Path())
	if err != nil {
		return err
	}

	expFeatures, err := lsctx.ExperimentalFeatures(ctx)
	if err != nil {
		return err
	}

	if expFeatures.ProcessJobsAsync {
		// Stop the timer from the previous request
		if jobTimer != nil {
			jobTimer.Stop()
		}

		// Set up a timer that will fire if no new didChange requests
		// were triggered in the specified duration.
		jobTimer = time.AfterFunc(startProcessingJobsAfter, func() {
			_, err = svc.indexer.DocumentChanged(dh.Dir)
			svc.logger.Printf("error scheduling jobs: %s", err)
		})

		// Return early without waiting on jobs
		return nil
	}

	jobIds, err := svc.indexer.DocumentChanged(dh.Dir)
	if err != nil {
		return err
	}

	return svc.stateStore.JobStore.WaitForJobs(ctx, jobIds...)
}
