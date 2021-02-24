package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/settings"
	"github.com/mitchellh/go-homedir"
)

func (lh *logHandler) Initialize(ctx context.Context, params lsp.InitializeParams) (lsp.InitializeResult, error) {
	serverCaps := lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: lsp.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    lsp.Incremental,
			},
			CompletionProvider: lsp.CompletionOptions{
				ResolveProvider: false,
			},
			HoverProvider:              true,
			DocumentFormattingProvider: true,
			DocumentSymbolProvider:     true,
			WorkspaceSymbolProvider:    true,
		},
	}

	serverCaps.ServerInfo.Name = "terraform-ls"
	version, ok := lsctx.LanguageServerVersion(ctx)
	if ok {
		serverCaps.ServerInfo.Version = version
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

	if params.ClientInfo.Name != "" {
		err = lsctx.SetClientName(ctx, params.ClientInfo.Name)
		if err != nil {
			return serverCaps, err
		}
	}

	clientCaps := params.Capabilities
	err = lsctx.SetClientCapabilities(ctx, &clientCaps)
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

	stCaps := clientCaps.TextDocument.SemanticTokens
	caps := ilsp.SemanticTokensClientCapabilities{
		SemanticTokensClientCapabilities: clientCaps.TextDocument.SemanticTokens,
	}
	semanticTokensOpts := lsp.SemanticTokensOptions{
		Legend: lsp.SemanticTokensLegend{
			TokenTypes:     ilsp.TokenTypesLegend(stCaps.TokenTypes).AsStrings(),
			TokenModifiers: ilsp.TokenModifiersLegend(stCaps.TokenModifiers).AsStrings(),
		},
		Full: caps.FullRequest(),
	}

	serverCaps.Capabilities.SemanticTokensProvider = semanticTokensOpts

	// set commandPrefix for session
	lsctx.SetCommandPrefix(ctx, out.Options.CommandPrefix)
	// apply prefix to executeCommand handler names
	serverCaps.Capabilities.ExecuteCommandProvider = lsp.ExecuteCommandOptions{
		Commands: handlers.Names(out.Options.CommandPrefix),
		WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{
			WorkDoneProgress: true,
		},
	}

	// set experimental feature flags
	lsctx.SetExperimentalFeatures(ctx, out.Options.ExperimentalFeatures)

	if len(out.UnusedKeys) > 0 {
		jrpc2.PushNotify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type:    lsp.Warning,
			Message: fmt.Sprintf("Unknown configuration options: %q", out.UnusedKeys),
		})
	}
	cfgOpts := out.Options

	// Static user-provided paths take precedence over dynamic discovery
	if len(cfgOpts.ModulePaths) > 0 {
		lh.logger.Printf("Attempting to add %d static module paths", len(cfgOpts.ModulePaths))
		for _, rawPath := range cfgOpts.ModulePaths {
			modPath, err := resolvePath(rootDir, rawPath)
			if err != nil {
				jrpc2.PushNotify(ctx, "window/showMessage", &lsp.ShowMessageParams{
					Type:    lsp.Warning,
					Message: fmt.Sprintf("Ignoring module path %s: %s", rawPath, err),
				})
				continue
			}

			err = w.AddModule(modPath)
			if err != nil {
				return serverCaps, err
			}
		}

		return serverCaps, nil
	}

	var excludeModulePaths []string
	for _, rawPath := range cfgOpts.ExcludeModulePaths {
		modPath, err := resolvePath(rootDir, rawPath)
		if err != nil {
			lh.logger.Printf("Ignoring excluded module path %s: %s", rawPath, err)
			continue
		}
		excludeModulePaths = append(excludeModulePaths, modPath)
	}

	walker, err := lsctx.ModuleWalker(ctx)
	if err != nil {
		return serverCaps, err
	}

	walker.SetLogger(lh.logger)
	walker.SetExcludeModulePaths(excludeModulePaths)
	// Walker runs asynchronously so we're intentionally *not*
	// passing the request context here
	bCtx := context.Background()
	err = walker.StartWalking(bCtx, fh.Dir())

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
