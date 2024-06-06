// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	"github.com/hashicorp/hcl-lang/lang"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) References(ctx context.Context, params lsp.ReferenceParams) ([]lsp.Location, error) {
	list := make([]lsp.Location, 0)

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return list, err
	}

	jobIds, err := svc.stateStore.JobStore.ListIncompleteJobsForDir(dh.Dir)
	if err != nil {
		return nil, err
	}
	svc.stateStore.JobStore.WaitForJobs(ctx, jobIds...)

	pos, err := ilsp.HCLPositionFromLspPosition(params.TextDocumentPositionParams.Position, doc)
	if err != nil {
		return list, err
	}

	path := lang.Path{
		Path:       doc.Dir.Path(),
		LanguageID: doc.LanguageID,
	}
	// TODO? maybe kick off indexing of the whole workspace here
	origins := svc.decoder.ReferenceOriginsTargetingPos(path, doc.Filename, pos)

	return ilsp.RefOriginsToLocations(origins), nil
}
