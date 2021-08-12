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
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
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
				ResolveProvider:   false,
				TriggerCharacters: []string{".", "["},
			},
			DeclarationProvider:        lsp.DeclarationOptions{},
			DefinitionProvider:         true,
			CodeLensProvider:           lsp.CodeLensOptions{},
			ReferencesProvider:         true,
			HoverProvider:              true,
			DocumentFormattingProvider: true,
			DocumentSymbolProvider:     true,
			WorkspaceSymbolProvider:    true,
			Workspace: lsp.Workspace5Gn{
				WorkspaceFolders: lsp.WorkspaceFolders4Gn{
					Supported:           true,
					ChangeNotifications: "workspace/didChangeWorkspaceFolders",
				},
			},
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

	if _, ok = lsp.ExperimentalClientCapabilities(clientCaps.Experimental).ShowReferencesCommandId(); ok {
		serverCaps.Capabilities.Experimental = lsp.ExperimentalServerCapabilities{
			ReferenceCountCodeLens: true,
		}
	}

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
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type:    lsp.Warning,
			Message: fmt.Sprintf("Unknown configuration options: %q", out.UnusedKeys),
		})
	}
	cfgOpts := out.Options

	// We might eventually remove cli flags for the following options
	path, ok := lsctx.TerraformExecPath(ctx)
	if len(path) > 0 && len(cfgOpts.TerraformExecPath) > 0 {
		return serverCaps, fmt.Errorf("Terraform exec path can either be set via (-tf-exec) CLI flag " +
			"or (terraformExecPath) LSP config option, not both")
	}

	var opts = &exec.ExecutorOpts{}
	if len(cfgOpts.TerraformExecPath) > 0 {
		opts.ExecPath = cfgOpts.TerraformExecPath
		ctx = exec.WithExecutorOpts(ctx, opts)
	}

	if !clientCaps.Workspace.WorkspaceFolders && len(params.WorkspaceFolders) > 0 {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: "Client sent workspace folders despite not declaring support. " +
				"Please report this as a bug.",
		})
	}

	walker, err := lsctx.ModuleWalker(ctx)
	if err != nil {
		return serverCaps, err
	}
	walker.SetLogger(lh.logger)

	var excludeModulePaths []string
	for _, rawPath := range cfgOpts.ExcludeModulePaths {
		modPath, err := resolvePath(rootDir, rawPath)
		if err != nil {
			lh.logger.Printf("Ignoring excluded module path %s: %s", rawPath, err)
			continue
		}
		excludeModulePaths = append(excludeModulePaths, modPath)
	}

	walker.SetExcludeModulePaths(excludeModulePaths)
	walker.EnqueuePath(fh.Dir())

	// Walker runs asynchronously so we're intentionally *not*
	// passing the request context here
	walkerCtx := context.Background()

	// Walker is also started early to allow gradual consumption
	// and avoid overfilling the queue
	err = walker.StartWalking(walkerCtx)
	if err != nil {
		return serverCaps, err
	}

	if len(params.WorkspaceFolders) > 0 {
		for _, folderPath := range params.WorkspaceFolders {
			modPath, err := pathFromDocumentURI(folderPath.URI)
			if err != nil {
				jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
					Type: lsp.Warning,
					Message: fmt.Sprintf("Ignoring workspace folder %s: %s."+
						" This is most likely bug, please report it.", folderPath.URI, err),
				})
				continue
			}

			walker.EnqueuePath(modPath)
		}
	}

	// Static user-provided paths take precedence over dynamic discovery
	if len(cfgOpts.ModulePaths) > 0 {
		lh.logger.Printf("Attempting to add %d static module paths", len(cfgOpts.ModulePaths))
		for _, rawPath := range cfgOpts.ModulePaths {
			modPath, err := resolvePath(rootDir, rawPath)
			if err != nil {
				jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
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

	return serverCaps, nil
}

func resolvePath(rootDir, rawPath string) (string, error) {
	path, err := homedir.Expand(rawPath)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(rootDir, rawPath)
	}

	return cleanupPath(path)
}

func cleanupPath(path string) (string, error) {
	absPath, err := filepath.EvalSymlinks(path)
	return toLowerVolumePath(absPath), err
}

func toLowerVolumePath(path string) string {
	volume := filepath.VolumeName(path)
	return strings.ToLower(volume) + path[len(volume):]
}
