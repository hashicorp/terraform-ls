package diagnostics

import (
	"context"
	"log"
	"path/filepath"
	"sync"

	"github.com/creachadair/jrpc2"
	"github.com/hashicorp/hcl/v2"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

type diagContext struct {
	ctx    context.Context
	uri    lsp.DocumentURI
	diags  hcl.Diagnostics
	source string
}

// Notifier is a type responsible for queueing hcl diagnostics to be converted
// and sent to the client
type Notifier struct {
	sessCtx        context.Context
	diags          chan diagContext
	closeDiagsOnce sync.Once
}

func NewNotifier(sessCtx context.Context, logger *log.Logger) *Notifier {
	diags := make(chan diagContext, 50)
	go notify(diags, logger)
	return &Notifier{sessCtx: sessCtx, diags: diags}
}

// Publish accepts a map of diagnostics per file and queues them for publishing
func (n *Notifier) Publish(ctx context.Context, rmDir string, diags map[string]hcl.Diagnostics, source string) {
	select {
	case <-n.sessCtx.Done():
		n.closeDiagsOnce.Do(func() {
			close(n.diags)
		})
		return
	default:
	}

	if source == "" {
		source = "Terraform"
	}

	for path, ds := range diags {
		n.diags <- diagContext{ctx: ctx, diags: ds, source: source, uri: lsp.DocumentURI(uri.FromPath(filepath.Join(rmDir, path)))}
	}
}

func notify(diags <-chan diagContext, logger *log.Logger) {
	for d := range diags {
		if err := jrpc2.PushNotify(d.ctx, "textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
			URI:         d.uri,
			Diagnostics: ilsp.HCLDiagsToLSP(d.diags, d.source),
		}); err != nil {
			logger.Printf("Error pushing diagnostics: %s", err)
		}
	}
}
