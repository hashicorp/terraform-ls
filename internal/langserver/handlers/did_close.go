// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) TextDocumentDidClose(ctx context.Context, params lsp.DidCloseTextDocumentParams) error {
	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	return svc.stateStore.DocumentStore.CloseDocument(dh)
}
