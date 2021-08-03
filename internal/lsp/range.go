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
			Character: uint32(fsRng.Start.Column),
			Line:      uint32(fsRng.Start.Line),
		},
		End: lsp.Position{
			Character: uint32(fsRng.End.Column),
			Line:      uint32(fsRng.End.Line),
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
		Start: HCLPosToLSP(rng.Start),
		End:   HCLPosToLSP(rng.End),
	}
}

func HCLPosToLSP(pos hcl.Pos) lsp.Position {
	return lsp.Position{
		Line:      uint32(pos.Line - 1),
		Character: uint32(pos.Column - 1),
	}
}
