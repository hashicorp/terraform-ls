package handlers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/sourcegraph/go-lsp"
)

func TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
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

	rootDir, _ := lsctx.RootDirectory(ctx)

	candidates := cf.RootModuleCandidatesByPath(f.Dir())
	if len(candidates) == 0 {
		msg := fmt.Sprintf("No root module found for %s."+
			" Functionality may be limited."+
			// Unfortunately we can't be any more specific wrt where
			// because we don't gather "init-able folders" in any way
			" You may need to run terraform init", f.Filename())
		return jrpc2.ServerPush(ctx, "window/showMessage", lsp.ShowMessageParams{
			Type:    lsp.MTWarning,
			Message: msg,
		})
	}
	if len(candidates) > 1 {
		// TODO: Suggest specifying explicit root modules?

		msg := fmt.Sprintf("Alternative root modules found for %s (%s), picked: %s",
			f.Filename(), renderCandidates(rootDir, candidates[1:]),
			renderCandidate(rootDir, candidates[0]))
		return jrpc2.ServerPush(ctx, "window/showMessage", lsp.ShowMessageParams{
			Type:    lsp.MTWarning,
			Message: msg,
		})
	}

	return nil
}

func renderCandidates(rootDir string, candidatePaths []string) string {
	for i, p := range candidatePaths {
		// This helps displaying shorter, but still relevant paths
		candidatePaths[i] = renderCandidate(rootDir, p)
	}
	return strings.Join(candidatePaths, ", ")
}

func renderCandidate(rootDir, path string) string {
	trimmed := strings.TrimPrefix(
		strings.TrimPrefix(path, rootDir), string(os.PathSeparator))
	if trimmed == "" {
		return "."
	}
	return trimmed
}
