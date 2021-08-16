package handlers

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

func TextDocumentDidClose(ctx context.Context, params lsp.DidCloseTextDocumentParams) error {
	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return err
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)
	err = fs.CloseAndRemoveDocument(fh)
	if err != nil {
		return err
	}

	if vf, ok := ast.NewVarsFilename(fh.Filename()); ok && !vf.IsAutoloaded() {
		notifier, err := lsctx.DiagnosticsNotifier(ctx)
		if err != nil {
			return err
		}

		diags := diagnostics.NewDiagnostics()
		diags.EmptyRootDiagnostic()
		diags.Append("HCL", map[string]hcl.Diagnostics{
			fh.Filename(): {},
		})
		notifier.PublishHCLDiags(ctx, fh.Dir(), diags)
	}

	return nil
}
