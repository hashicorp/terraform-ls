package handlers

import (
	"context"
	"fmt"
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
		msg := fmt.Sprintf("Alternative root modules found for %s:\n%s",
			f.Filename(), strings.Join(candidates, "\n"))
		return jrpc2.ServerPush(ctx, "window/showMessage", lsp.ShowMessageParams{
			Type:    lsp.MTWarning,
			Message: msg,
		})
	}

	return nil
}
