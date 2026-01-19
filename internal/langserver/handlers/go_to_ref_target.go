// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) GoToDefinition(ctx context.Context, params lsp.TextDocumentPositionParams) (interface{}, error) {
	cc, err := ilsp.ClientCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	targets, err := svc.goToReferenceTarget(ctx, params)
	if err != nil {
		return nil, err
	}

	return ilsp.RefTargetsToDefinitionLocationLinks(targets, cc.TextDocument.Definition), nil
}

func (svc *service) GoToDeclaration(ctx context.Context, params lsp.TextDocumentPositionParams) (interface{}, error) {
	cc, err := ilsp.ClientCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	targets, err := svc.goToReferenceTarget(ctx, params)
	if err != nil {
		return nil, err
	}

	return ilsp.RefTargetsToDeclarationLocationLinks(targets, cc.TextDocument.Declaration), nil
}

func (svc *service) goToReferenceTarget(ctx context.Context, params lsp.TextDocumentPositionParams) (decoder.ReferenceTargets, error) {
	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return nil, err
	}

	pos, err := ilsp.HCLPositionFromLspPosition(params.Position, doc)
	if err != nil {
		return nil, err
	}

	jobIds, err := svc.stateStore.JobStore.ListIncompleteJobsForDir(dh.Dir)
	if err != nil {
		return nil, err
	}
	svc.stateStore.JobStore.WaitForJobs(ctx, jobIds...)

	path := lang.Path{
		Path:       doc.Dir.Path(),
		LanguageID: doc.LanguageID,
	}

	return svc.decoder.ReferenceTargetsForOriginAtPos(path, doc.Filename, pos)
}
