// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func HCLSeverityToLSP(severity hcl.DiagnosticSeverity) lsp.DiagnosticSeverity {
	var sev lsp.DiagnosticSeverity
	switch severity {
	case hcl.DiagError:
		sev = lsp.SeverityError
	case hcl.DiagWarning:
		sev = lsp.SeverityWarning
	case hcl.DiagInvalid:
		panic("invalid diagnostic")
	}
	return sev
}

func HCLDiagsToLSP(hclDiags hcl.Diagnostics, source string) []lsp.Diagnostic {
	diags := []lsp.Diagnostic{}

	for _, hclDiag := range hclDiags {
		msg := hclDiag.Summary
		if hclDiag.Detail != "" {
			msg += ": " + hclDiag.Detail
		}
		var rnge lsp.Range
		if hclDiag.Subject != nil {
			rnge = HCLRangeToLSP(*hclDiag.Subject)
		}

		diag := lsp.Diagnostic{
			Range:    rnge,
			Severity: HCLSeverityToLSP(hclDiag.Severity),
			Source:   source,
			Message:  msg,
		}

		if code, ok := hclDiag.Extra.(hclsyntax.CodeDiagExtra); ok {
			diag.Code = code
		}

		diags = append(diags, diag)

	}
	return diags
}

func LSPDiagsToHCL(lsplDiags []lsp.Diagnostic, filename string) hcl.Diagnostics {
	diags := hcl.Diagnostics{}

	for _, lspDiag := range lsplDiags {
		parts := strings.Split(lspDiag.Message, ":")
		summary := parts[0]
		detail := ""
		if len(parts) > 1 {
			detail = strings.Join(parts[1:], ":")
		}

		diag := &hcl.Diagnostic{
			Severity: LSPDiagSeverityToHCL(lspDiag.Severity),
			Summary:  summary,
			Detail:   detail,
			Subject:  LSPRangeToHCL(lspDiag.Range, filename).Ptr(),
		}

		if code, ok := lspDiag.Code.(string); ok && code != "" {
			diag.Extra = hclsyntax.CodeDiagExtra(code)
		}

		diags = append(diags, diag)
	}

	return diags
}

func LSPDiagSeverityToHCL(severity lsp.DiagnosticSeverity) hcl.DiagnosticSeverity {
	var sev hcl.DiagnosticSeverity
	switch severity {
	case lsp.SeverityError:
		sev = hcl.DiagError
	case lsp.SeverityWarning:
		sev = hcl.DiagWarning
	default:
		panic(fmt.Sprintf("unexpected diagnostic severity %q", severity))
	}
	return sev
}
