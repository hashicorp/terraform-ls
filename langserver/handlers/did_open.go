package handlers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	lsp "github.com/sourcegraph/go-lsp"
)

func (lh *logHandler) TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return err
	}

	f := ilsp.FileFromDocumentItem(params.TextDocument)
	err = fs.Open(f)
	if err != nil {
		return err
	}

	cf, err := lsctx.RootModuleCandidateFinder(ctx)
	if err != nil {
		return err
	}

	walker, err := lsctx.RootModuleWalker(ctx)
	if err != nil {
		return err
	}

	rootDir, _ := lsctx.RootDirectory(ctx)
	dir := relativePath(rootDir, f.Dir())

	candidates := cf.RootModuleCandidatesByPath(f.Dir())

	if walker.IsWalking() {
		// avoid raising false warnings if walker hasn't finished yet
		lh.logger.Printf("walker has not finished walking yet, data may be inaccurate for %s", f.FullPath())
	} else if len(candidates) == 0 {
		// TODO: Only notify once per f.Dir() per session
		msg := fmt.Sprintf("No root module found for %s."+
			" Functionality may be limited."+
			// Unfortunately we can't be any more specific wrt where
			// because we don't gather "init-able folders" in any way
			" You may need to run terraform init"+
			" and reload your editor.", dir)
		return jrpc2.ServerPush(ctx, "window/showMessage", lsp.ShowMessageParams{
			Type:    lsp.MTWarning,
			Message: msg,
		})
	}
	if len(candidates) > 1 {
		msg := fmt.Sprintf("Alternative root modules found for %s (%s), picked: %s."+
			" You can try setting paths to root modules explicitly in settings.",
			dir, candidatePaths(rootDir, candidates[1:]),
			relativePath(rootDir, candidates[0].Path()))
		return jrpc2.ServerPush(ctx, "window/showMessage", lsp.ShowMessageParams{
			Type:    lsp.MTWarning,
			Message: msg,
		})
	}

	return nil
}

func candidatePaths(rootDir string, candidates []rootmodule.RootModule) string {
	paths := make([]string, len(candidates))
	for i, rm := range candidates {
		// This helps displaying shorter, but still relevant paths
		paths[i] = relativePath(rootDir, rm.Path())
	}
	return strings.Join(paths, ", ")
}

func relativePath(rootDir string, path string) string {
	trimmed := strings.TrimPrefix(
		strings.TrimPrefix(path, rootDir), string(os.PathSeparator))
	if trimmed == "" {
		return "."
	}
	return trimmed
}
