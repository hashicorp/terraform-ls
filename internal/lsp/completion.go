package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/mdplain"
	lsp "github.com/sourcegraph/go-lsp"
)

func CompletionList(candidates lang.Candidates, caps lsp.TextDocumentClientCapabilities) lsp.CompletionList {
	list := lsp.CompletionList{
		Items:        make([]lsp.CompletionItem, len(candidates.List)),
		IsIncomplete: !candidates.IsComplete,
	}

	snippetSupport := caps.Completion.CompletionItem.SnippetSupport

	markdown := false
	docsFormat := caps.Completion.CompletionItem.DocumentationFormat
	if len(docsFormat) > 0 && docsFormat[0] == "markdown" {
		markdown = true
	}

	for i, c := range candidates.List {
		list.Items[i] = CompletionItem(c, snippetSupport, markdown)
	}

	return list
}

func CompletionItem(candidate lang.Candidate, snippet, markdown bool) lsp.CompletionItem {
	doc := candidate.Description.Value

	// TODO: revisit once go-lsp supports markdown in CompletionItem
	doc = mdplain.Clean(doc)

	var kind lsp.CompletionItemKind
	switch candidate.Kind {
	case lang.AttributeCandidateKind:
		kind = lsp.CIKProperty
	case lang.BlockCandidateKind:
		kind = lsp.CIKClass
	case lang.LabelCandidateKind:
		kind = lsp.CIKField
	}

	te, format := textEdit(candidate.TextEdit, snippet)

	return lsp.CompletionItem{
		Label:            candidate.Label,
		Kind:             kind,
		InsertTextFormat: format,
		Detail:           candidate.Detail,
		Documentation:    doc,
		TextEdit:         te,
	}
}

func textEdit(te lang.TextEdit, snippetSupport bool) (*lsp.TextEdit, lsp.InsertTextFormat) {
	if snippetSupport {
		return &lsp.TextEdit{
			NewText: te.Snippet,
			Range:   HCLRangeToLSP(te.Range),
		}, lsp.ITFSnippet
	}

	return &lsp.TextEdit{
		NewText: te.NewText,
		Range:   HCLRangeToLSP(te.Range),
	}, lsp.ITFPlainText
}
