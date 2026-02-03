// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package diagnostics

import (
	"context"
	"log"
	"path/filepath"
	"sync"

	"github.com/hashicorp/hcl/v2"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

type diagContext struct {
	ctx   context.Context
	uri   lsp.DocumentURI
	diags []lsp.Diagnostic
}

type ClientNotifier interface {
	Notify(ctx context.Context, method string, params interface{}) error
}

// Notifier is a type responsible for queueing HCL diagnostics to be converted
// and sent to the client
type Notifier struct {
	logger         *log.Logger
	diags          chan diagContext
	clientNotifier ClientNotifier
	closeDiagsOnce sync.Once
}

func NewNotifier(clientNotifier ClientNotifier, logger *log.Logger) *Notifier {
	n := &Notifier{
		logger:         logger,
		diags:          make(chan diagContext, 50),
		clientNotifier: clientNotifier,
	}
	go n.notify()
	return n
}

// PublishHCLDiags accepts a map of HCL diagnostics per file and queues them for publishing.
// A dir path is passed which is joined with the filename keys of the map, to form a file URI.
func (n *Notifier) PublishHCLDiags(ctx context.Context, dirPath string, diags Diagnostics) {
	select {
	case <-ctx.Done():
		n.closeDiagsOnce.Do(func() {
			close(n.diags)
		})
		return
	default:
	}

	for filename, ds := range diags {
		fileDiags := make([]lsp.Diagnostic, 0)
		for source, diags := range ds {
			fileDiags = append(fileDiags, ilsp.HCLDiagsToLSP(diags, source.String())...)
		}

		n.diags <- diagContext{
			ctx:   ctx,
			uri:   lsp.DocumentURI(uri.FromPath(filepath.Join(dirPath, filename))),
			diags: fileDiags,
		}
	}
}

func (n *Notifier) notify() {
	for d := range n.diags {
		if err := n.clientNotifier.Notify(d.ctx, "textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
			URI:         d.uri,
			Diagnostics: d.diags,
		}); err != nil {
			n.logger.Printf("Error pushing diagnostics: %s", err)
		}
	}
}

type Diagnostics map[string]map[ast.DiagnosticSource]hcl.Diagnostics

func NewDiagnostics() Diagnostics {
	return make(Diagnostics, 0)
}

// EmptyRootDiagnostic allows emptying any diagnostics for
// the whole directory which were published previously.
func (d Diagnostics) EmptyRootDiagnostic() Diagnostics {
	d[""] = make(map[ast.DiagnosticSource]hcl.Diagnostics, 0)
	return d
}

func (d Diagnostics) Append(src ast.DiagnosticSource, diagsMap map[string]hcl.Diagnostics) Diagnostics {
	for uri, uriDiags := range diagsMap {
		if _, ok := d[uri]; !ok {
			d[uri] = make(map[ast.DiagnosticSource]hcl.Diagnostics, 0)
		}
		d[uri][src] = uriDiags
	}

	return d
}

func (d Diagnostics) Extend(diags Diagnostics) Diagnostics {
	for uri, uriDiags := range diags {
		if _, ok := d[uri]; !ok {
			d[uri] = make(map[ast.DiagnosticSource]hcl.Diagnostics, 0)
		}
		for src, diag := range uriDiags {
			d[uri][src] = append(d[uri][src], diag...)
		}
	}

	return d
}
