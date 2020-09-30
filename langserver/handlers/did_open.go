package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	lsp "github.com/sourcegraph/go-lsp"
)

func (lh *logHandler) TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {

	diags, err := lsctx.Diagnostics(ctx)
	if err != nil {
		return err
	}
	diags.DiagnoseHCL(ctx, params.TextDocument.URI, []byte(params.TextDocument.Text))

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return err
	}

	f := ilsp.FileFromDocumentItem(params.TextDocument)
	err = fs.CreateAndOpenDocument(f, f.Text())
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
	readableDir := humanReadablePath(rootDir, f.Dir())
	candidates := cf.RootModuleCandidatesByPath(f.Dir())

	if walker.IsWalking() {
		// avoid raising false warnings if walker hasn't finished yet
		lh.logger.Printf("walker has not finished walking yet, data may be inaccurate for %s", f.FullPath())
	} else if len(candidates) == 0 {
		// TODO: Only notify once per f.Dir() per session
		msg := fmt.Sprintf("No root module found for %q."+
			" Functionality may be limited."+
			// Unfortunately we can't be any more specific wrt where
			// because we don't gather "init-able folders" in any way
			" You may need to run terraform init"+
			" and reload your editor.", readableDir)
		return jrpc2.PushNotify(ctx, "window/showMessage", lsp.ShowMessageParams{
			Type:    lsp.MTWarning,
			Message: msg,
		})
	}
	if len(candidates) > 1 {
		candidateDir := humanReadablePath(rootDir, candidates[0].Path())

		msg := fmt.Sprintf("Alternative root modules found for %s (%s), picked: %s."+
			" You can try setting paths to root modules explicitly in settings.",
			readableDir, candidatePaths(rootDir, candidates[1:]),
			candidateDir)
		return jrpc2.PushNotify(ctx, "window/showMessage", lsp.ShowMessageParams{
			Type:    lsp.MTWarning,
			Message: msg,
		})
	}

	return nil
}

func candidatePaths(rootDir string, candidates []rootmodule.RootModule) string {
	paths := make([]string, len(candidates))
	for i, rm := range candidates {
		paths[i] = humanReadablePath(rootDir, rm.Path())
	}
	return strings.Join(paths, ", ")
}

// humanReadablePath helps displaying shorter, but still relevant paths
func humanReadablePath(rootDir, path string) string {
	if rootDir == "" {
		return path
	}

	// absolute paths can be too long for UI/messages,
	// so we just display relative to root dir
	relDir, err := filepath.Rel(rootDir, path)
	if err != nil {
		return path
	}

	if relDir == "." {
		// Name of the root dir is more helpful than "."
		return filepath.Base(rootDir)
	}

	return relDir
}
