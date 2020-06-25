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
		msg := fmt.Sprintf("No root module found for %s"+
			" functionality may be limited", f.Filename())
		return jrpc2.ServerPush(ctx, "window/showMessage", lsp.ShowMessageParams{
			Type:    lsp.MTWarning,
			Message: msg,
		})
	}
	if len(candidates) > 1 {
		// TODO: Suggest specifying explicit root modules?

		msg := fmt.Sprintf("Alternative root modules found for %s (%s), picked: %s",
			f.Filename(), candidatePaths(rootDir, candidates[1:]),
			renderCandidatePath(rootDir, candidates[0]))
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
		paths[i] = renderCandidatePath(rootDir, rm)
	}
	return strings.Join(paths, ", ")
}

func renderCandidatePath(rootDir string, candidate rootmodule.RootModule) string {
	trimmed := strings.TrimPrefix(
		strings.TrimPrefix(candidate.Path(), rootDir), string(os.PathSeparator))
	if trimmed == "" {
		return "."
	}
	return trimmed
}
