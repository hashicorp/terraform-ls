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
	// TODO: deprecated / tags?

	doc := ""
	if c := candidate.Documentation(); c != nil {
		// TODO: markdown handling
		doc = c.Value()
	}

	if snippetSupport {
		return lsp.CompletionItem{
			Label:            candidate.Label(),
			Kind:             lsp.CIKField,
			InsertTextFormat: lsp.ITFSnippet,
			Detail:           candidate.Detail(),
			Documentation:    doc,
			TextEdit:         textEdit(candidate.Snippet(), pos),
		}
	}

	return lsp.CompletionItem{
		Label:            candidate.Label(),
		Kind:             lsp.CIKField,
		InsertTextFormat: lsp.ITFPlainText,
		Detail:           candidate.Detail(),
		Documentation:    doc,
		TextEdit:         textEdit(candidate.PlainText(), pos),
	}
}

func textEdit(te lang.TextEdit, pos hcl.Pos) *lsp.TextEdit {
	rng := te.Range()
	if rng == nil {
		rng = &hcl.Range{
			Start: pos,
			End:   pos,
		}
	}

	return &lsp.TextEdit{
		NewText: te.NewText(),
		Range:   hclRangeToLSP(*rng),
	}
}
