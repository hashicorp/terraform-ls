// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func HoverData(data *lang.HoverData, cc lsp.TextDocumentClientCapabilities) *lsp.Hover {
	if data == nil {
		return nil
	}
	mdSupported := len(cc.Hover.ContentFormat) > 0 &&
		cc.Hover.ContentFormat[0] == "markdown"

	// In theory we should be sending lsp.MarkedString (for old clients)
	// when len(cc.Hover.ContentFormat) == 0, but that's not possible
	// without changing lsp.Hover.Content field type to interface{}
	//
	// We choose to follow gopls' approach (i.e. cut off old clients).

	return &lsp.Hover{
		Contents: markupContent(data.Content, mdSupported),
		Range:    HCLRangeToLSP(data.Range),
	}
}
