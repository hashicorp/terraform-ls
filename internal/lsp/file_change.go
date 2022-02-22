package lsp

import (
	"github.com/hashicorp/terraform-ls/internal/document"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

type contentChange struct {
	text string
	rng  *document.Range
}

func ContentChange(chEvent lsp.TextDocumentContentChangeEvent) document.Change {
	return &contentChange{
		text: chEvent.Text,
		rng:  lspRangeToDocRange(chEvent.Range),
	}
}

func DocumentChanges(events []lsp.TextDocumentContentChangeEvent) document.Changes {
	changes := make(document.Changes, len(events))
	for i, event := range events {
		ch := ContentChange(event)
		changes[i] = ch
	}
	return changes
}

func (fc *contentChange) Text() string {
	return fc.text
}

func (fc *contentChange) Range() *document.Range {
	return fc.rng
}
