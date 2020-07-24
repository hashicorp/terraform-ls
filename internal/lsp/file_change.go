package lsp

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/sourcegraph/go-lsp"
)

type contentChange struct {
	text string
	rng  hcl.Range
}

func ContentChange(chEvent lsp.TextDocumentContentChangeEvent, f File) (*contentChange, error) {
	if chEvent.Range != nil {
		rng, err := lspRangeToHCL(*chEvent.Range, f)
		if err != nil {
			return nil, err
		}

		return &contentChange{
			text: chEvent.Text,
			rng:  *rng,
		}, nil
	}

	return &contentChange{
		text: chEvent.Text,
	}, nil
}

func DocumentChanges(events []lsp.TextDocumentContentChangeEvent, f File) (filesystem.DocumentChanges, error) {
	changes := make(filesystem.DocumentChanges, len(events))
	for i, event := range events {
		ch, err := ContentChange(event, f)
		if err != nil {
			return nil, err
		}
		changes[i] = ch
	}
	return changes, nil
}

func TextEdits(changes filesystem.DocumentChanges) []lsp.TextEdit {
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

func lspRangeToHCL(lspRng lsp.Range, f File) (*hcl.Range, error) {
	startPos, err := lspPositionToHCL(f.Lines(), lspRng.Start)
	if err != nil {
		return nil, err
	}

	endPos, err := lspPositionToHCL(f.Lines(), lspRng.End)
	if err != nil {
		return nil, err
	}

	return &hcl.Range{
		Filename: f.Filename(),
		Start:    startPos,
		End:      endPos,
	}, nil
}

func (fc *contentChange) Text() string {
	return fc.text
}

func (fc *contentChange) Range() hcl.Range {
	return fc.rng
}
