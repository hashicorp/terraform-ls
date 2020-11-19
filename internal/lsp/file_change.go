package lsp

import (
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

type contentChange struct {
	text string
	rng  *filesystem.Range
}

func ContentChange(chEvent lsp.TextDocumentContentChangeEvent) filesystem.DocumentChange {
	return &contentChange{
		text: chEvent.Text,
		rng:  lspRangeToFsRange(chEvent.Range),
	}
}

func DocumentChanges(events []lsp.TextDocumentContentChangeEvent, f File) (filesystem.DocumentChanges, error) {
	changes := make(filesystem.DocumentChanges, len(events))
	for i, event := range events {
		ch := ContentChange(event)
		changes[i] = ch
	}
	return changes, nil
}

func (fc *contentChange) Text() string {
	return fc.text
}

func (fc *contentChange) Range() *filesystem.Range {
	return fc.rng
}
