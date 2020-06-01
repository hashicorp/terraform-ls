package lsp

import (
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
	lsp "github.com/sourcegraph/go-lsp"
)

func CompletionList(candidates lang.CompletionCandidates, caps lsp.TextDocumentClientCapabilities) lsp.CompletionList {
	snippetSupport := caps.Completion.CompletionItem.SnippetSupport
	list := lsp.CompletionList{}

	if candidates == nil {
		return list
	}

	cList := candidates.List()

	list.IsIncomplete = !candidates.IsComplete()
	list.Items = make([]lsp.CompletionItem, len(cList))
	for i, c := range cList {
		list.Items[i] = CompletionItem(c, snippetSupport)
	}

	return list
}

func CompletionItem(candidate lang.CompletionCandidate, snippetSupport bool) lsp.CompletionItem {
	// TODO: deprecated / tags?

	doc := ""
	if c := candidate.Documentation(); c != nil {
		// TODO: markdown handling
		doc = c.Value()
	}

	r := candidate.PrefixRange()
	if snippetSupport {
		return lsp.CompletionItem{
			Label:            candidate.Label(),
			Kind:             lsp.CIKField,
			InsertTextFormat: lsp.ITFSnippet,
			Detail:           candidate.Detail(),
			Documentation:    doc,
			TextEdit: &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: r.Start.Line - 1, Character: r.Start.Column - 1},
					End:   lsp.Position{Line: r.End.Line - 1, Character: r.End.Column - 1},
				},
				NewText: candidate.Snippet(),
			},
		}
	}

	return lsp.CompletionItem{
		Label:            candidate.Label(),
		Kind:             lsp.CIKField,
		InsertTextFormat: lsp.ITFPlainText,
		Detail:           candidate.Detail(),
		Documentation:    doc,
		TextEdit: &lsp.TextEdit{
			Range: lsp.Range{
				Start: lsp.Position{Line: r.Start.Line - 1, Character: r.Start.Column - 1},
				End:   lsp.Position{Line: r.End.Line - 1, Character: r.End.Column - 1},
			},
			NewText: candidate.Label(),
		},
	}
}
