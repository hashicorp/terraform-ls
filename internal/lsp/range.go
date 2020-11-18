package lsp

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	lsp "github.com/sourcegraph/go-lsp"
)

func fsRangeToLSP(fsRng *filesystem.Range) lsp.Range {
	if fsRng == nil {
		return lsp.Range{}
	}

	return lsp.Range{
		Start: lsp.Position{
			Character: fsRng.Start.Column,
			Line:      fsRng.Start.Line,
		},
		End: lsp.Position{
			Character: fsRng.End.Column,
			Line:      fsRng.End.Line,
		},
	}
}

func lspRangeToFsRange(rng *lsp.Range) *filesystem.Range {
	if rng == nil {
		return nil
	}

	return &filesystem.Range{
		Start: filesystem.Pos{
			Line:   rng.Start.Line,
			Column: rng.Start.Character,
		},
		End: filesystem.Pos{
			Line:   rng.End.Line,
			Column: rng.End.Character,
		},
	}
}

func HCLRangeToLSP(rng hcl.Range) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{
			Line:      rng.Start.Line - 1,
			Character: rng.Start.Column - 1,
		},
		End: lsp.Position{
			Line:      rng.End.Line - 1,
			Character: rng.End.Column - 1,
		},
	}
}
