// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/mdplain"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) CompletionItemResolve(ctx context.Context, params lsp.CompletionItemWithResolveHook) (lsp.CompletionItemWithResolveHook, error) {
	cc, err := ilsp.ClientCapabilities(ctx)
	if err != nil {
		return params, err
	}

	if params.ResolveHook == nil {
		return params, nil
	}

	unresolvedCandidate := decoder.UnresolvedCandidate{
		ResolveHook: params.ResolveHook,
	}

	resolvedCandidate, err := svc.decoder.ResolveCandidate(ctx, unresolvedCandidate)
	if err != nil || resolvedCandidate == nil {
		return params, err
	}

	if resolvedCandidate.Description.Value != "" {
		doc := resolvedCandidate.Description.Value

		// TODO: Revisit when MarkupContent is allowed as Documentation
		// https://github.com/golang/tools/blob/4783bc9b/internal/lsp/protocol/tsprotocol.go#L753
		doc = mdplain.Clean(doc)
		params.Documentation = doc
	}
	if resolvedCandidate.Detail != "" {
		params.Detail = resolvedCandidate.Detail
	}
	if len(resolvedCandidate.AdditionalTextEdits) > 0 {
		snippetSupport := cc.TextDocument.Completion.CompletionItem.SnippetSupport
		params.AdditionalTextEdits = ilsp.TextEdits(resolvedCandidate.AdditionalTextEdits, snippetSupport)
	}

	return params, nil
}
