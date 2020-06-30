package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
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
	if fh.URI() == "" || !fh.IsDir() {
		return serverCaps, fmt.Errorf("Editing a single file is not yet supported." +
			" Please open a directory.")
	}
	if !fh.Valid() {
		return serverCaps, fmt.Errorf("URI %q is not valid", params.RootURI)
	}

	err := lsctx.SetRootDirectory(ctx, fh.FullPath())
	if err != nil {
		return serverCaps, err
	}

	err = lsctx.SetClientCapabilities(ctx, &params.Capabilities)
	if err != nil {
		return serverCaps, err
	}

	rmm, err := lsctx.RootModuleManager(ctx)
	if err != nil {
		return serverCaps, err
	}

	w, err := lsctx.Watcher(ctx)
	if err != nil {
		return serverCaps, err
	}

	walker, err := lsctx.RootModuleWalker(ctx)
	if err != nil {
		return serverCaps, err
	}

	walker.SetLogger(lh.logger)
	err = walker.WalkInitializedRootModules(fh.Dir(), func(dir string) error {
		lh.logger.Printf("Adding root module: %s", dir)
		rm, err := rmm.AddRootModule(dir)
		if err != nil {
			return err
		}

		paths := rm.PathsToWatch()
		lh.logger.Printf("Adding %d paths of root module for watching (%s)", len(paths), dir)
		err = w.AddPaths(paths)
		if err != nil {
			return err
		}

		return nil
	})

	return serverCaps, err
}
