package lsp

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func fsRangeToLSP(fsRng *filesystem.Range) lsp.Range {
	if fsRng == nil {
		return lsp.Range{}
	}

	return lsp.Range{
		Start: lsp.Position{
			Character: float64(fsRng.Start.Column),
			Line:      float64(fsRng.Start.Line),
		},
		End: lsp.Position{
			Character: float64(fsRng.End.Column),
			Line:      float64(fsRng.End.Line),
		},
	}
}

func lspRangeToFsRange(rng *lsp.Range) *filesystem.Range {
	if rng == nil {
		return nil
	}

	return &filesystem.Range{
		Start: filesystem.Pos{
			Line:   int(rng.Start.Line),
			Column: int(rng.Start.Character),
		},
		End: filesystem.Pos{
			Line:   int(rng.End.Line),
			Column: int(rng.End.Character),
		},
	}
}

func HCLRangeToLSP(rng hcl.Range) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{
			Line:      float64(rng.Start.Line - 1),
			Character: float64(rng.Start.Column - 1),
		},
		End: lsp.Position{
			Line:      float64(rng.End.Line - 1),
			Character: float64(rng.End.Column - 1),
		},
	}
}
