// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) TextDocumentSymbol(ctx context.Context, params lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	var symbols []lsp.DocumentSymbol

	cc, err := ilsp.ClientCapabilities(ctx)
	if err != nil {
		return symbols, err
	}

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return symbols, err
	}

	jobIds, err := svc.stateStore.JobStore.ListIncompleteJobsForDir(dh.Dir)
	if err != nil {
		return symbols, err
	}
	svc.stateStore.JobStore.WaitForJobs(ctx, jobIds...)

	d, err := svc.decoderForDocument(ctx, doc)
	if err != nil {
		return symbols, err
	}

	sbs, err := d.SymbolsInFile(doc.Filename)
	if err != nil {
		return symbols, err
	}

	return ilsp.DocumentSymbols(sbs, cc.TextDocument.DocumentSymbol), nil
}
