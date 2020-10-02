package lsp

import (
	"github.com/hashicorp/hcl/v2"
	lsp "github.com/sourcegraph/go-lsp"
)

func HCLRangeToLSP(hclRng hcl.Range) lsp.Range {
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

func HCLSeverityToLSP(severity hcl.DiagnosticSeverity) lsp.DiagnosticSeverity {
	var sev lsp.DiagnosticSeverity
	switch severity {
	case hcl.DiagError:
		sev = lsp.Error
	case hcl.DiagWarning:
		sev = lsp.Warning
	case hcl.DiagInvalid:
		panic("invalid diagnostic")
	}
	return sev
}
