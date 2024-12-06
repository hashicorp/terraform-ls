// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	"github.com/hashicorp/hcl-lang/lang"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) TextDocumentCodeLens(ctx context.Context, params lsp.CodeLensParams) ([]lsp.CodeLens, error) {
	list := make([]lsp.CodeLens, 0)

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return list, err
	}

	jobIds, err := svc.stateStore.JobStore.ListIncompleteJobsForDir(dh.Dir)
	if err != nil {
		return list, err
	}
	svc.stateStore.JobStore.WaitForJobs(ctx, jobIds...)

	path := lang.Path{
		Path:       doc.Dir.Path(),
		LanguageID: doc.LanguageID,
		File:       doc.Filename,
	}

	lenses, err := svc.decoder.CodeLensesForFile(ctx, path, doc.Filename)
	if err != nil {
		return nil, err
	}

	for _, lens := range lenses {
		cmd, err := ilsp.Command(lens.Command)
		if err != nil {
			svc.logger.Printf("skipping code lens %#v: %s", lens.Command, err)
			continue
		}

		list = append(list, lsp.CodeLens{
			Range:   ilsp.HCLRangeToLSP(lens.Range),
			Command: cmd,
		})
	}

	return list, nil
}
