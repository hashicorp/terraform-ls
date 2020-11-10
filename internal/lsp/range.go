package lsp

import (
	"github.com/hashicorp/hcl/v2"
	lsp "github.com/sourcegraph/go-lsp"
)

func HCLRangeToLSP(hclRng hcl.Range) lsp.Range {
	r := lsp.Range{
		Start: lsp.Position{
			Character: hclRng.Start.Column - 1,
			Line:      hclRng.Start.Line - 1,
		},
		End: lsp.Position{
			Character: hclRng.End.Column - 1,
			Line:      hclRng.End.Line - 1,
		},
	}

	if r.Start.Character < 0 {
		r.Start.Character = 0
	}
	if r.Start.Line < 0 {
		r.Start.Line = 0
	}
	if r.End.Character < 0 {
		r.End.Character = 0
	}
	if r.End.Line < 0 {
		r.End.Line = 0
	}

	return r
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
