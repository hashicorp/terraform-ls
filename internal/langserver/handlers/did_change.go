package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func TextDocumentDidChange(ctx context.Context, params lsp.DidChangeTextDocumentParams) error {
	p := lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: params.TextDocument.URI,
			},
			Version: params.TextDocument.Version,
		},
		ContentChanges: params.ContentChanges,
	}

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return err
	}

	fh := ilsp.VersionedFileHandler(p.TextDocument)
	f, err := fs.GetDocument(fh)
	if err != nil {
		return err
	}

	// Versions don't have to be consecutive, but they must be increasing
	if int(p.TextDocument.Version) <= f.Version() {
		fs.CloseAndRemoveDocument(fh)
		return fmt.Errorf("Old version (%d) received, current version is %d. "+
			"Unable to update %s. This is likely a bug, please report it.",
			int(p.TextDocument.Version), f.Version(), p.TextDocument.URI)
	}

	changes, err := ilsp.DocumentChanges(params.ContentChanges, f)
	if err != nil {
		return err
	}
	err = fs.ChangeDocument(fh, changes)
	if err != nil {
		return err
	}

	modMgr, err := lsctx.ModuleManager(ctx)
	if err != nil {
		return err
	}

	mod, err := modMgr.ModuleByPath(fh.Dir())
	if err != nil {
		return err
	}

	err = modMgr.EnqueueModuleOpWait(mod.Path(), module.OpTypeParseConfiguration)
	if err != nil {
		return err
	}

	diags, err := lsctx.Diagnostics(ctx)
	if err != nil {
		return err
	}
	diags.PublishHCLDiags(ctx, mod.Path(), mod.Diagnostics(), "HCL")

	return nil
}
