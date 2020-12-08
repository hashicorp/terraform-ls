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
	source string
	diags  []lsp.Diagnostic
}

// Notifier is a type responsible for queueing hcl diagnostics to be converted
// and sent to the client
type Notifier struct {
	logger         *log.Logger
	sessCtx        context.Context
	diags          chan diagContext
	diagsCache     map[string]map[string][]lsp.Diagnostic
	closeDiagsOnce sync.Once
}

func NewNotifier(sessCtx context.Context, logger *log.Logger) *Notifier {
	n := &Notifier{
		logger:     logger,
		sessCtx:    sessCtx,
		diags:      make(chan diagContext, 50),
		diagsCache: make(map[string]map[string][]lsp.Diagnostic),
	}
	go n.notify()
	return n
}

// PublishHCLDiags accepts a map of hcl diagnostics per file and queues them for publishing.
// A dir path is passed which is joined with the filename keys of the map, to form a file URI.
// A source string is passed and set for each diagnostic, this is typically displayed in the client UI.
func (n *Notifier) PublishHCLDiags(ctx context.Context, dirPath string, diags map[string]hcl.Diagnostics, source string) {
	select {
	case <-n.sessCtx.Done():
		n.closeDiagsOnce.Do(func() {
			close(n.diags)
		})
		return
	default:
	}

	for filename, ds := range diags {
		n.diags <- diagContext{
			ctx: ctx, source: source,
			diags: ilsp.HCLDiagsToLSP(ds, source),
			uri:   lsp.DocumentURI(uri.FromPath(filepath.Join(dirPath, filename))),
		}
	}
}

func (n *Notifier) notify() {
	for d := range n.diags {
		if err := jrpc2.PushNotify(d.ctx, "textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
			URI:         d.uri,
			Diagnostics: n.mergeDiags(string(d.uri), d.source, d.diags),
		}); err != nil {
			n.logger.Printf("Error pushing diagnostics: %s", err)
		}
	}
}

// mergeDiags will return all diags from all cached sources for a given uri.
// the passed diags overwrites the cached entry for the passed source key
// even if empty
func (n *Notifier) mergeDiags(uri string, source string, diags []lsp.Diagnostic) []lsp.Diagnostic {
	fileDiags, ok := n.diagsCache[uri]
	if !ok {
		fileDiags = make(map[string][]lsp.Diagnostic)
	}

	fileDiags[source] = diags
	n.diagsCache[uri] = fileDiags

	all := []lsp.Diagnostic{}
	for _, diags := range fileDiags {
		all = append(all, diags...)
	}
	return all
}
