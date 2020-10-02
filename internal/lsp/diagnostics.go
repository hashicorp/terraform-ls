package lsp

import (
	"github.com/hashicorp/hcl/v2"
	lsp "github.com/sourcegraph/go-lsp"
)

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
