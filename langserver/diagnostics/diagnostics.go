package diagnostics

import (
	"context"
	"sync"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/sourcegraph/go-lsp"
)

type documentContext struct {
	ctx  context.Context
	uri  lsp.DocumentURI
	text []byte
}

type Notifier struct {
	sessCtx          context.Context
	hclDocs          chan<- documentContext
	closeHclDocsOnce sync.Once
}

func NewNotifier(sessCtx context.Context) *Notifier {
	hclDocs := make(chan documentContext, 10)
	go hclDiags(hclDocs)
	return &Notifier{hclDocs: hclDocs, sessCtx: sessCtx}
}

func (n *Notifier) DiagnoseHCL(ctx context.Context, uri lsp.DocumentURI, text []byte) {
	select {
	case <-n.sessCtx.Done():
		n.closeHclDocsOnce.Do(func() {
			close(n.hclDocs)
		})
		return
	case n.hclDocs <- documentContext{ctx: ctx, uri: uri, text: text}:
	}
}

func hclDiags(docs <-chan documentContext) {
	for d := range docs {
		diags := []lsp.Diagnostic{}

		_, hclDiags := hclparse.NewParser().ParseHCL(d.text, string(d.uri))
		for _, hclDiag := range hclDiags {
			if hclDiag.Subject != nil {
				diags = append(diags, lsp.Diagnostic{
					Range:    ilsp.HCLRangeToLSP(*hclDiag.Subject),
					Severity: hclSeverityToLSP(hclDiag.Severity),
					Source:   "HCL",
					Message:  lspMessage(hclDiag),
				})
			}
		}

		if err := jrpc2.PushNotify(d.ctx, "textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
			URI:         d.uri,
			Diagnostics: diags,
		}); !acceptableError(err) {
			panic(err)
		}
	}
}

func hclSeverityToLSP(severity hcl.DiagnosticSeverity) lsp.DiagnosticSeverity {
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

func lspMessage(diag *hcl.Diagnostic) string {
	m := diag.Summary
	if diag.Detail != "" {
		m += ": " + diag.Detail
	}
	return m
}

func acceptableError(err error) bool {
	return true
}
