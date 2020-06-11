package handlers

import (
	"context"
	"errors"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	tferr "github.com/hashicorp/terraform-ls/internal/terraform/errors"
	lsp "github.com/sourcegraph/go-lsp"
)

func (lh *logHandler) Initialize(ctx context.Context, params lsp.InitializeParams) (lsp.InitializeResult, error) {
	serverCaps := lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
				Options: &lsp.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    lsp.TDSKIncremental,
				},
			},
			CompletionProvider: &lsp.CompletionOptions{
				ResolveProvider: false,
			},
			DocumentFormattingProvider: true,
		},
	}

	fh := ilsp.FileHandlerFromDirURI(params.RootURI)
	if !fh.Valid() {
		return serverCaps, fmt.Errorf("URI %q is not valid", params.RootURI)
	}

	if !fh.IsDir() {
		return serverCaps, fmt.Errorf("Editing a single file is not yet supported." +
			" Please open a directory.")
	}

	err := lsctx.SetClientCapabilities(ctx, &params.Capabilities)
	if err != nil {
		return serverCaps, err
	}

	wm, err := lsctx.RootModuleManager(ctx)
	if err != nil {
		return serverCaps, err
	}

	ww, err := lsctx.Watcher(ctx)
	if err != nil {
		return serverCaps, err
	}

	err = wm.AddRootModule(fh.Dir())
	if err != nil {
		if errors.Is(err, &tferr.NotInitializedErr{}) {
			return serverCaps, fmt.Errorf("Directory not initialized. "+
				"Please run `terraform init` in %s", fh.Dir())
		}
		return serverCaps, err
	}

	err = ww.AddPaths(wm.PathsToWatch())
	if err != nil {
		return serverCaps, err
	}

	err = ww.Start()
	if err != nil {
		return serverCaps, err
	}

	return serverCaps, nil
}
