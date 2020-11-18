package lsp

import (
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/sourcegraph/go-lsp"
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

func TextEdits(changes filesystem.DocumentChanges) []lsp.TextEdit {
	edits := make([]lsp.TextEdit, len(changes))

	for i, change := range changes {
		edits[i] = lsp.TextEdit{
			Range:   fsRangeToLSP(change.Range()),
			NewText: change.Text(),
		}
	}

	return edits
}

func (fc *contentChange) Text() string {
	return fc.text
}

func (fc *contentChange) Range() *filesystem.Range {
	return fc.rng
}
