package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentDidChange(ctx context.Context, params lsp.DidChangeTextDocumentParams) error {
	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return err
	}

	fh := ilsp.VersionedFileHandler(params.TextDocument)
	f, err := fs.GetFile(fh)
	if err != nil {
		return err
	}
	changes, err := ilsp.FileChanges(params.ContentChanges, f)
	if err != nil {
		return err
	}
	return fs.Change(fh, changes)
}
