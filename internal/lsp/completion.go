package lsp

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
	lsp "github.com/sourcegraph/go-lsp"
)

func CompletionList(candidates lang.CompletionCandidates, pos hcl.Pos, caps lsp.TextDocumentClientCapabilities) lsp.CompletionList {
	snippetSupport := caps.Completion.CompletionItem.SnippetSupport
	list := lsp.CompletionList{}

	if candidates == nil {
		return list
	}

	cList := candidates.List()

	list.IsIncomplete = !candidates.IsComplete()
	list.Items = make([]lsp.CompletionItem, len(cList))
	for i, c := range cList {
		list.Items[i] = CompletionItem(c, pos, snippetSupport)
	}

	return list
}

func CompletionItem(candidate lang.CompletionCandidate, pos hcl.Pos, snippetSupport bool) lsp.CompletionItem {
	if snippetSupport {
		pos, newText := candidate.Snippet(pos)

		return lsp.CompletionItem{
			Label:            candidate.Label(),
			Kind:             lsp.CIKField,
			InsertTextFormat: lsp.ITFSnippet,
			Detail:           candidate.Detail(),
			TextEdit: &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
					End:   lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
				},
				NewText: newText,
			},
		}
	}

	return lsp.CompletionItem{
		Label:            candidate.Label(),
		Kind:             lsp.CIKField,
		InsertTextFormat: lsp.ITFPlainText,
		Detail:           candidate.Detail(),
	}
}
