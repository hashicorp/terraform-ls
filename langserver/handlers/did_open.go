package handlers

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/internal/watcher"
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

	rmm, err := lsctx.RootModuleManager(ctx)
	if err != nil {
		return err
	}

	walker, err := lsctx.RootModuleWalker(ctx)
	if err != nil {
		return err
	}

	w, err := lsctx.Watcher(ctx)
	if err != nil {
		return err
	}

	rootDir, _ := lsctx.RootDirectory(ctx)
	readableDir := humanReadablePath(rootDir, f.Dir())

	var rm rootmodule.RootModule

	rm, err = rmm.RootModuleByPath(f.Dir())
	if err != nil {
		if rootmodule.IsRootModuleNotFound(err) {
			rm, err = rmm.AddAndStartLoadingRootModule(ctx, f.Dir())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lh.logger.Printf("opened root module: %s", rm.Path())

	// We reparse because the file being opened may not match
	// (originally parsed) content on the disk
	// TODO: Do this only if we can verify the file differs?
	err = rm.ParseFiles()
	if err != nil {
		return fmt.Errorf("failed to parse files: %w", err)
	}

	// TOOD: Publish diags from c.ParsedDiagnostics()

	candidates := rmm.RootModuleCandidatesByPath(f.Dir())

	if walker.IsWalking() {
		// avoid raising false warnings if walker hasn't finished yet
		lh.logger.Printf("walker has not finished walking yet, data may be inaccurate for %s", f.FullPath())
	} else if len(candidates) == 0 {
		// TODO: Only notify once per f.Dir() per session
		// TODO: if there is any change for the provider/module, ask for init. The did change handler should also implement this
		if !rm.ParsedDiagnostics().HasErrors() && len(f.Text()) != 0 {
			go askInitForEmptyRootModule(ctx, lh.logger, w, rootDir, f.Dir())
		}
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

func askInitForEmptyRootModule(ctx context.Context, logger *log.Logger, w watcher.Watcher, rootDir, dir string) {
	msg := fmt.Sprintf("No root module found for %q."+
		" Functionality may be limited."+
		// Unfortunately we can't be any more specific wrt where
		// because we don't gather "init-able folders" in any way
		" You may need to run terraform init.", humanReadablePath(rootDir, dir))
	title := "run `terraform init`"
	resp, err := jrpc2.PushCall(ctx, "window/showMessageRequest", lsp.ShowMessageRequestParams{
		Type:    lsp.MTWarning,
		Message: msg,
		Actions: []lsp.MessageActionItem{
			{
				Title: title,
			},
		},
	})
	if err != nil {
		logger.Printf("%+v", err)
		return
	}
	var action lsp.MessageActionItem
	if err := resp.UnmarshalResult(&action); err != nil {
		logger.Printf("unmarshal MessageActionItem: %+v", err)
		return
	}
	if action.Title == title {
		rmm, err := lsctx.RootModuleManager(ctx)
		if err != nil {
			logger.Printf("%+v", err)
			return
		}

		rm, err := rmm.InitAndUpdateRootModule(ctx, dir)
		if err != nil {
			logger.Printf("failed to init root module %+v", err)
			return
		}

		paths := rm.PathsToWatch()
		logger.Printf("Adding %d paths of root module for watching (%s)", len(paths), dir)
		err = w.AddPaths(paths)
		if err != nil {
			logger.Printf("failed to add watch for dir (%s): %+v", dir, err)
		}
	}
}
