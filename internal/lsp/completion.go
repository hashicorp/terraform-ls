package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/mdplain"
	lsp "github.com/sourcegraph/go-lsp"
)

type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

type CompletionItem struct {
	lsp.CompletionItem
	Command *lsp.Command `json:"command,omitempty"`
}

func ToCompletionList(candidates lang.Candidates, caps lsp.TextDocumentClientCapabilities) CompletionList {
	list := CompletionList{
		Items:        make([]CompletionItem, len(candidates.List)),
		IsIncomplete: !candidates.IsComplete,
	}

	snippetSupport := caps.Completion.CompletionItem.SnippetSupport

	markdown := false
	docsFormat := caps.Completion.CompletionItem.DocumentationFormat
	if len(docsFormat) > 0 && docsFormat[0] == "markdown" {
		markdown = true
	}

	for i, c := range candidates.List {
		list.Items[i] = toCompletionItem(c, snippetSupport, markdown)
	}

	return list
}

func toCompletionItem(candidate lang.Candidate, snippet, markdown bool) CompletionItem {
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
	var cmd *lsp.Command
	if candidate.TriggerSuggest {
		cmd = &lsp.Command{
			Command: "editor.action.triggerSuggest",
			Title:   "Suggest",
		}
	}

	return CompletionItem{
		CompletionItem: lsp.CompletionItem{
			Label:            candidate.Label,
			Kind:             kind,
			InsertTextFormat: format,
			Detail:           candidate.Detail,
			Documentation:    doc,
			TextEdit:         te,
		},
		Command: cmd,
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
