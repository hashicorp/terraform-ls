package lsp

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/document"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func documentRangeToLSP(docRng *document.Range) lsp.Range {
	if docRng == nil {
		return lsp.Range{}
	}

	return lsp.Range{
		Start: lsp.Position{
			Character: uint32(docRng.Start.Column),
			Line:      uint32(docRng.Start.Line),
		},
		End: lsp.Position{
			Character: uint32(docRng.End.Column),
			Line:      uint32(docRng.End.Line),
		},
	}
}

func lspRangeToDocRange(rng *lsp.Range) *document.Range {
	if rng == nil {
		return nil
	}

	return &document.Range{
		Start: document.Pos{
			Line:   int(rng.Start.Line),
			Column: int(rng.Start.Character),
		},
		End: document.Pos{
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
