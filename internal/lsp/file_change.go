package lsp

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/sourcegraph/go-lsp"
)

type fileChange struct {
	text string
	rng  hcl.Range
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

func TextEdits(changes filesystem.FileChanges) []lsp.TextEdit {
	edits := make([]lsp.TextEdit, len(changes))

	for i, change := range changes {
		edits[i] = lsp.TextEdit{
			Range:   hclRangeToLSP(change.Range()),
			NewText: change.Text(),
		}
	}

	return edits
}

func hclRangeToLSP(hclRng hcl.Range) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{
			Character: hclRng.Start.Column - 1,
			Line:      hclRng.Start.Line - 1,
		},
		End: lsp.Position{
			Character: hclRng.End.Column - 1,
			Line:      hclRng.End.Line - 1,
		},
	}
}

func (fc *fileChange) Text() string {
	return fc.text
}

func (fc *fileChange) Range() hcl.Range {
	return fc.rng
}
