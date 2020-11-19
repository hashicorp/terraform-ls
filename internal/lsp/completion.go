package lsp

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/mdplain"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func ToCompletionList(candidates lang.Candidates, caps lsp.TextDocumentClientCapabilities) lsp.CompletionList {
	list := lsp.CompletionList{
		Items:        make([]lsp.CompletionItem, len(candidates.List)),
		IsIncomplete: !candidates.IsComplete,
	}

	snippetSupport := caps.Completion.CompletionItem.SnippetSupport

	markdown := false
	docsFormat := caps.Completion.CompletionItem.DocumentationFormat
	if len(docsFormat) > 0 && docsFormat[0] == lsp.Markdown {
		markdown = true
	}

	for i, c := range candidates.List {
		list.Items[i] = toCompletionItem(c, snippetSupport, markdown)
	}

	return list
}

func toCompletionItem(candidate lang.Candidate, snippet, markdown bool) lsp.CompletionItem {
	doc := candidate.Description.Value

	// TODO: Revisit when MarkupContent is allowed as Documentation
	// https://github.com/golang/tools/blob/4783bc9b/internal/lsp/protocol/tsprotocol.go#L753
	doc = mdplain.Clean(doc)

	var kind lsp.CompletionItemKind
	switch candidate.Kind {
	case lang.AttributeCandidateKind:
		kind = lsp.PropertyCompletion
	case lang.BlockCandidateKind:
		kind = lsp.ClassCompletion
	case lang.LabelCandidateKind:
		kind = lsp.FieldCompletion
	}

	var cmd *lsp.Command
	if candidate.TriggerSuggest {
		cmd = &lsp.Command{
			Command: "editor.action.triggerSuggest",
			Title:   "Suggest",
		}
	}

	return lsp.CompletionItem{
		Label:               candidate.Label,
		Kind:                kind,
		InsertTextFormat:    insertTextFormat(snippet),
		Detail:              candidate.Detail,
		Documentation:       doc,
		TextEdit:            textEdit(candidate.TextEdit, snippet),
		Command:             cmd,
		AdditionalTextEdits: textEdits(candidate.AdditionalTextEdits, snippet),
	}
}
