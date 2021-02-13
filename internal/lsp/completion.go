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

	for i, c := range candidates.List {
		list.Items[i] = toCompletionItem(c, caps.Completion)
	}

	return list
}

func toCompletionItem(candidate lang.Candidate, caps lsp.CompletionClientCapabilities) lsp.CompletionItem {
	snippetSupport := caps.CompletionItem.SnippetSupport

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
	case lang.BoolCandidateKind:
		kind = lsp.EnumMemberCompletion
	case lang.StringCandidateKind:
		kind = lsp.TextCompletion
	case lang.NumberCandidateKind:
		kind = lsp.ValueCompletion
	case lang.KeywordCandidateKind:
		kind = lsp.KeywordCompletion
	case lang.ListCandidateKind, lang.SetCandidateKind, lang.TupleCandidateKind:
		kind = lsp.EnumCompletion
	case lang.MapCandidateKind, lang.ObjectCandidateKind:
		kind = lsp.StructCompletion
	}

	// TODO: Omit item which uses kind unsupported by the client

	var cmd *lsp.Command
	if candidate.TriggerSuggest {
		cmd = &lsp.Command{
			Command: "editor.action.triggerSuggest",
			Title:   "Suggest",
		}
	}

	item := lsp.CompletionItem{
		Label:               candidate.Label,
		Kind:                kind,
		InsertTextFormat:    insertTextFormat(snippetSupport),
		Detail:              candidate.Detail,
		Documentation:       doc,
		TextEdit:            textEdit(candidate.TextEdit, snippetSupport),
		Command:             cmd,
		AdditionalTextEdits: textEdits(candidate.AdditionalTextEdits, snippetSupport),
	}

	if caps.CompletionItem.DeprecatedSupport {
		item.Deprecated = candidate.IsDeprecated
	}
	if tagSliceContains(caps.CompletionItem.TagSupport.ValueSet,
		lsp.ComplDeprecated) && candidate.IsDeprecated {
		item.Tags = []lsp.CompletionItemTag{
			lsp.ComplDeprecated,
		}
	}

	return item
}

func tagSliceContains(supported []lsp.CompletionItemTag, tag lsp.CompletionItemTag) bool {
	for _, item := range supported {
		if item == tag {
			return true
		}
	}
	return false
}
