package lsp

import (
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/sourcegraph/go-lsp"
)

type fileChange struct {
	text string
}

func FileChange(chEvent lsp.TextDocumentContentChangeEvent, f File) (*fileChange, error) {
	if chEvent.Range != nil {
		return nil, fmt.Errorf("Partial updates are not supported (yet)")
	}

	return &fileChange{
		text: chEvent.Text,
	}, nil
}

func FileChanges(events []lsp.TextDocumentContentChangeEvent, f File) (filesystem.FileChanges, error) {
	changes := make(filesystem.FileChanges, len(events))
	for i, event := range events {
		ch, err := FileChange(event, f)
		if err != nil {
			return nil, err
		}
		changes[i] = ch
	}
	return changes, nil
}

func (fc *fileChange) Text() string {
	return fc.text
}
