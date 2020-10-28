package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/settings"
	"github.com/mitchellh/go-homedir"
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
			DocumentSymbolProvider:     true,
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

	rootDir := fh.FullPath()
	err := lsctx.SetRootDirectory(ctx, rootDir)
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

	addAndLoadRootModule, err := lsctx.RootModuleLoader(ctx)
	if err != nil {
		return serverCaps, err
	}

	w, err := lsctx.Watcher(ctx)
	if err != nil {
		return serverCaps, err
	}

	out, err := settings.DecodeOptions(params.InitializationOptions)
	if err != nil {
		return serverCaps, err
	}
	err = out.Options.Validate()
	if err != nil {
		return serverCaps, err
	}

	// set server ID
	err = lsctx.SetServerID(ctx, out.Options.ID)
	if err != nil {
		return serverCaps, err
	}
	// apply suffix to executeCommand handler names
	serverCaps.Capabilities.ExecuteCommandProvider = &lsp.ExecuteCommandOptions{
		Commands: handlers.Names(out.Options.ID),
	}
	if len(out.UnusedKeys) > 0 {
		jrpc2.PushNotify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type:    lsp.MTWarning,
			Message: fmt.Sprintf("Unknown configuration options: %q", out.UnusedKeys),
		})
	}
	cfgOpts := out.Options

	// Static user-provided paths take precedence over dynamic discovery
	if len(cfgOpts.RootModulePaths) > 0 {
		lh.logger.Printf("Attempting to add %d static root module paths", len(cfgOpts.RootModulePaths))
		for _, rawPath := range cfgOpts.RootModulePaths {
			rmPath, err := resolvePath(rootDir, rawPath)
			if err != nil {
				jrpc2.PushNotify(ctx, "window/showMessage", &lsp.ShowMessageParams{
					Type:    lsp.MTWarning,
					Message: fmt.Sprintf("Ignoring root module path %s: %s", rawPath, err),
				})
				continue
			}
			rm, err := addAndLoadRootModule(rmPath)
			if err != nil {
				return serverCaps, err
			}

			paths := rm.PathsToWatch()
			lh.logger.Printf("Adding %d paths of root module for watching (%s)", len(paths), rmPath)
			err = w.AddPaths(paths)
			if err != nil {
				return serverCaps, err
			}
		}

		return serverCaps, nil
	}

	var excludeModulePaths []string
	for _, rawPath := range cfgOpts.ExcludeModulePaths {
		rmPath, err := resolvePath(rootDir, rawPath)
		if err != nil {
			lh.logger.Printf("Ignoring exclude root module path %s: %s", rawPath, err)
			continue
		}
		excludeModulePaths = append(excludeModulePaths, rmPath)
	}

	walker, err := lsctx.RootModuleWalker(ctx)
	if err != nil {
		return serverCaps, err
	}

	walker.SetLogger(lh.logger)
	walker.SetExcludeModulePaths(excludeModulePaths)
	// Walker runs asynchronously so we're intentionally *not*
	// passing the request context here
	bCtx := context.Background()
	err = walker.StartWalking(bCtx, fh.Dir(), func(ctx context.Context, dir string) error {
		lh.logger.Printf("Adding root module: %s", dir)
		rm, err := rmm.AddAndStartLoadingRootModule(ctx, dir)
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

func resolvePath(rootDir, rawPath string) (string, error) {
	path, err := homedir.Expand(rawPath)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(rootDir, rawPath)
	}

	absPath, err := filepath.EvalSymlinks(path)
	return toLowerVolumePath(absPath), err
}

func toLowerVolumePath(path string) string {
	volume := filepath.VolumeName(path)
	return strings.ToLower(volume) + path[len(volume):]
}
