package diagnostics

import (
	"context"
	"strings"
	"sync"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/hcl/v2/hclparse"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/sourcegraph/go-lsp"
)

// documentContext encapsulates the data needed to diagnose the file and push diagnostics to the client
type documentContext struct {
	ctx  context.Context
	uri  lsp.DocumentURI
	text []byte
}

// Notifier is a type responsible for processing documents and pushing diagnostics to the client
type Notifier struct {
	sessCtx          context.Context
	hclDocs          chan documentContext
	closeHclDocsOnce sync.Once
}

func NewNotifier(sessCtx context.Context) *Notifier {
	hclDocs := make(chan documentContext, 10)
	go hclDiags(hclDocs)
	return &Notifier{hclDocs: hclDocs, sessCtx: sessCtx}
}

// DiagnoseHCL enqueues the document for HCL parsing. Documents will be parsed and notifications delivered in order that
// they are enqueued. Files that are actively changing should be enqueued in order, so that diagnostics remain insync with
// the current content of the file. This is the responsibility of the caller.
func (n *Notifier) DiagnoseHCL(ctx context.Context, uri lsp.DocumentURI, text []byte) {
	select {
	case <-n.sessCtx.Done():
		n.closeHclDocsOnce.Do(func() {
			close(n.hclDocs)
		})
		return
	default:
	}
	n.hclDocs <- documentContext{ctx: ctx, uri: uri, text: text}
}

func hclParse(doc documentContext) []lsp.Diagnostic {
	diags := []lsp.Diagnostic{}

	_, hclDiags := hclparse.NewParser().ParseHCL(doc.text, string(doc.uri))
	for _, hclDiag := range hclDiags {
		// only process diagnostics with an attributable spot in the code
		if hclDiag.Subject != nil {
			msg := hclDiag.Summary
			if hclDiag.Detail != "" {
				msg += ": " + hclDiag.Detail
			}
			diags = append(diags, lsp.Diagnostic{
				Range:    ilsp.HCLRangeToLSP(*hclDiag.Subject),
				Severity: ilsp.HCLSeverityToLSP(hclDiag.Severity),
				Source:   "HCL",
				Message:  msg,
			})
		}
	}
	return diags
}

func hclDiags(docs <-chan documentContext) {
	for doc := range docs {
		// always push diagnostics, even if the slice is empty, this is how previous diagnostics are cleared
		// any push error will result in a panic since this is executing in its own thread and we can't bubble
		// an error to a jrpc response
		if err := jrpc2.PushNotify(doc.ctx, "textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
			URI:         doc.uri,
			Diagnostics: hclParse(doc),
		}); fatalError(err) {
			panic(err)
		}
	}
}

func fatalError(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "client connection is closed") {
		return false
	}
	return true
}
