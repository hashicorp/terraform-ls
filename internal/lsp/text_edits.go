// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/document"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func TextEditsFromDocumentChanges(changes document.Changes) []lsp.TextEdit {
	edits := make([]lsp.TextEdit, len(changes))

	for i, change := range changes {
		edits[i] = lsp.TextEdit{
			Range:   documentRangeToLSP(change.Range()),
			NewText: change.Text(),
		}
	}

	return edits
}

func TextEdits(tes []lang.TextEdit, snippetSupport bool) []lsp.TextEdit {
	edits := make([]lsp.TextEdit, len(tes))

	for i, te := range tes {
		edits[i] = *textEdit(te, snippetSupport)
	}

	return edits
}

func textEdit(te lang.TextEdit, snippetSupport bool) *lsp.TextEdit {
	if snippetSupport {
		return &lsp.TextEdit{
			NewText: te.Snippet,
			Range:   HCLRangeToLSP(te.Range),
		}
	}

	return &lsp.TextEdit{
		NewText: te.NewText,
		Range:   HCLRangeToLSP(te.Range),
	}
}

func insertTextFormat(snippetSupport bool) lsp.InsertTextFormat {
	if snippetSupport {
		return lsp.SnippetTextFormat
	}

	return lsp.PlainTextTextFormat
}
