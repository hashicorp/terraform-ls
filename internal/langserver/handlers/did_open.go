package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/creachadair/jrpc2"
	"github.com/google/uuid"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/handlers/command"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func (lh *logHandler) TextDocumentDidOpen(ctx context.Context, params lsp.DidOpenTextDocumentParams) error {
	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return err
	}

	f := ilsp.FileFromDocumentItem(params.TextDocument)
	err = fs.CreateAndOpenDocument(f, f.Text())
	if err != nil {
		return err
	}

	modMgr, err := lsctx.ModuleManager(ctx)
	if err != nil {
		return err
	}

	walker, err := lsctx.ModuleWalker(ctx)
	if err != nil {
		return err
	}

	rootDir, _ := lsctx.RootDirectory(ctx)
	readableDir := humanReadablePath(rootDir, f.Dir())

	var mod module.Module

	mod, err = modMgr.ModuleByPath(f.Dir())
	if err != nil {
		if module.IsModuleNotFound(err) {
			mod, err = modMgr.AddAndStartLoadingModule(ctx, f.Dir())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lh.logger.Printf("opened module: %s", mod.Path())

	// We reparse because the file being opened may not match
	// (originally parsed) content on the disk
	// TODO: Do this only if we can verify the file differs?
	err = mod.ParseFiles()
	if err != nil {
		return fmt.Errorf("failed to parse files: %w", err)
	}

	diags, err := lsctx.Diagnostics(ctx)
	if err != nil {
		return err
	}
	diags.PublishHCLDiags(ctx, mod.Path(), mod.ParsedDiagnostics(), "HCL")

	candidates := modMgr.ModuleCandidatesByPath(f.Dir())

	if walker.IsWalking() {
		// avoid raising false warnings if walker hasn't finished yet
		lh.logger.Printf("walker has not finished walking yet, data may be inaccurate for %s", f.FullPath())
	} else if len(candidates) == 0 {
		// TODO: Only notify once per f.Dir() per session
		dh := ilsp.FileHandlerFromDirPath(f.Dir())
		go func() {
			err := askInitForNoModuleCandidates(ctx, rootDir, dh)
			if err != nil {
				jrpc2.PushNotify(ctx, "window/showMessage", lsp.ShowMessageParams{
					Type:    lsp.Error,
					Message: err.Error(),
				})
			}
		}()
	}

	if len(candidates) > 1 {
		candidateDir := humanReadablePath(rootDir, candidates[0].Path())

		msg := fmt.Sprintf("Alternative schema source found for %s (%s), picked: %s."+
			" You can set an explicit module path in your settings.",
			readableDir, candidatePaths(rootDir, candidates[1:]),
			candidateDir)
		return jrpc2.PushNotify(ctx, "window/showMessage", lsp.ShowMessageParams{
			Type:    lsp.Warning,
			Message: msg,
		})
	}

	return nil
}

func candidatePaths(rootDir string, candidates []module.Module) string {
	paths := make([]string, len(candidates))
	for i, mod := range candidates {
		paths[i] = humanReadablePath(rootDir, mod.Path())
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

func askInitForNoModuleCandidates(ctx context.Context, rootDir string, dh ilsp.DirHandler) error {
	msg := fmt.Sprintf("No schema found for %q."+
		" Functionality may be limited."+
		// Unfortunately we can't be any more specific wrt where
		// because we don't gather "init-able folders" in any way
		" You may need to run terraform init.", humanReadablePath(rootDir, dh.Dir()))
	title := "terraform init"
	resp, err := jrpc2.PushCall(ctx, "window/showMessageRequest", lsp.ShowMessageRequestParams{
		Type:    lsp.Info,
		Message: msg,
		Actions: []lsp.MessageActionItem{
			{
				Title: title,
			},
		},
	})
	if err != nil {
		return err
	}
	var action lsp.MessageActionItem
	if err := resp.UnmarshalResult(&action); err != nil {
		return fmt.Errorf("unmarshal MessageActionItem: %+v", err)
	}
	if action.Title == title {
		ctx, err := initiateProgress(ctx)
		if err != nil {
			return err
		}
		_, err = command.TerraformInitHandler(ctx, cmd.CommandArgs{
			"uri": dh.URI(),
		})
		if err != nil {
			return fmt.Errorf("Initialization failed: %w", err)
		}
		return nil
	}
	return nil
}

func initiateProgress(ctx context.Context) (context.Context, error) {
	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return ctx, err
	}

	if !cc.Window.WorkDoneProgress {
		// server-side reporting not supported
		return ctx, nil
	}

	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	token := lsp.ProgressToken(id.String())

	_, err = jrpc2.PushCall(ctx, "window/workDoneProgress/create", lsp.WorkDoneProgressCreateParams{
		Token: token,
	})
	if err == nil {
		return lsctx.WithProgressToken(ctx, token), nil
	}

	return ctx, err
}
