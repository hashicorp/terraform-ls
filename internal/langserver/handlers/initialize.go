package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/settings"
	"github.com/hashicorp/terraform-ls/internal/uri"
	"github.com/mitchellh/go-homedir"
)

func (svc *service) Initialize(ctx context.Context, params lsp.InitializeParams) (lsp.InitializeResult, error) {
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
			CodeActionProvider: lsp.CodeActionOptions{
				CodeActionKinds: ilsp.SupportedCodeActions.AsSlice(),
				ResolveProvider: false,
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

	clientCaps := params.Capabilities

	properties := map[string]interface{}{
		"experimentalCapabilities.referenceCountCodeLens": false,
		"options.rootModulePaths":                         false,
		"options.excludeModulePaths":                      false,
		"options.commandPrefix":                           false,
		"options.ignoreDirectoryNames":                    false,
		"options.experimentalFeatures.validateOnSave":     false,
		"options.terraformExecPath":                       false,
		"options.terraformExecTimeout":                    "",
		"options.terraformLogFilePath":                    false,
		"root_uri":                                        "dir",
		"lsVersion":                                       serverCaps.ServerInfo.Version,
	}

	expClientCaps := lsp.ExperimentalClientCapabilities(clientCaps.Experimental)

	svc.server = jrpc2.ServerFromContext(ctx)

	if tv, ok := expClientCaps.TelemetryVersion(); ok {
		svc.logger.Printf("enabling telemetry (version: %d)", tv)
		err := svc.setupTelemetry(tv, svc.server)
		if err != nil {
			svc.logger.Printf("failed to setup telemetry: %s", err)
		}
		svc.logger.Printf("telemetry enabled (version: %d)", tv)
	}
	defer svc.telemetry.SendEvent(ctx, "initialize", properties)

	if params.RootURI == "" {
		properties["root_uri"] = "file"
		return serverCaps, fmt.Errorf("Editing a single file is not yet supported." +
			" Please open a directory.")
	}
	if !uri.IsURIValid(string(params.RootURI)) {
		properties["root_uri"] = "invalid"
		return serverCaps, fmt.Errorf("URI %q is not valid", params.RootURI)
	}

	root := document.DirHandleFromURI(string(params.RootURI))

	err := lsctx.SetRootDirectory(ctx, root.Path())
	if err != nil {
		return serverCaps, err
	}

	if params.ClientInfo.Name != "" {
		err = ilsp.SetClientName(ctx, params.ClientInfo.Name)
		if err != nil {
			return serverCaps, err
		}
	}

	if _, ok = expClientCaps.ShowReferencesCommandId(); ok {
		serverCaps.Capabilities.Experimental = lsp.ExperimentalServerCapabilities{
			ReferenceCountCodeLens: true,
		}
		properties["experimentalCapabilities.referenceCountCodeLens"] = true
	}

	err = ilsp.SetClientCapabilities(ctx, &clientCaps)
	if err != nil {
		return serverCaps, err
	}

	out, err := settings.DecodeOptions(params.InitializationOptions)
	if err != nil {
		return serverCaps, err
	}

	properties["options.rootModulePaths"] = len(out.Options.ModulePaths) > 0
	properties["options.excludeModulePaths"] = len(out.Options.ExcludeModulePaths) > 0
	properties["options.commandPrefix"] = len(out.Options.CommandPrefix) > 0
	properties["options.ignoreDirectoryNames"] = len(out.Options.IgnoreDirectoryNames) > 0
	properties["options.experimentalFeatures.prefillRequiredFields"] = out.Options.ExperimentalFeatures.PrefillRequiredFields
	properties["options.experimentalFeatures.validateOnSave"] = out.Options.ExperimentalFeatures.ValidateOnSave
	properties["options.terraformExecPath"] = len(out.Options.TerraformExecPath) > 0
	properties["options.terraformExecTimeout"] = out.Options.TerraformExecTimeout
	properties["options.terraformLogFilePath"] = len(out.Options.TerraformLogFilePath) > 0

	err = out.Options.Validate()
	if err != nil {
		return serverCaps, err
	}

	err = svc.configureSessionDependencies(ctx, out.Options)
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
		Commands: cmdHandlers(svc).Names(out.Options.CommandPrefix),
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

	if !clientCaps.Workspace.WorkspaceFolders && len(params.WorkspaceFolders) > 0 {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: "Client sent workspace folders despite not declaring support. " +
				"Please report this as a bug.",
		})
	}

	var excludeModulePaths []string
	for _, rawPath := range cfgOpts.ExcludeModulePaths {
		modPath, err := resolvePath(root.Path(), rawPath)
		if err != nil {
			svc.logger.Printf("Ignoring excluded module path %s: %s", rawPath, err)
			continue
		}
		excludeModulePaths = append(excludeModulePaths, modPath)
	}

	svc.walker.SetIgnoreDirectoryNames(cfgOpts.IgnoreDirectoryNames)
	svc.walker.SetExcludeModulePaths(excludeModulePaths)
	svc.walker.EnqueuePath(root.Path())

	if len(params.WorkspaceFolders) > 0 {
		for _, folderPath := range params.WorkspaceFolders {
			modPath, err := uri.PathFromURI(folderPath.URI)
			if err != nil {
				jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
					Type: lsp.Warning,
					Message: fmt.Sprintf("Ignoring workspace folder %s: %s."+
						" This is most likely bug, please report it.", folderPath.URI, err),
				})
				continue
			}

			svc.walker.EnqueuePath(modPath)
		}
	}

	// Walker runs asynchronously so we're intentionally *not*
	// passing the request context here
	walkerCtx := context.Background()
	err = svc.walker.StartWalking(walkerCtx)
	if err != nil {
		return serverCaps, err
	}

	// Static user-provided paths take precedence over dynamic discovery
	if len(cfgOpts.ModulePaths) > 0 {
		svc.logger.Printf("Attempting to add %d static module paths", len(cfgOpts.ModulePaths))
		for _, rawPath := range cfgOpts.ModulePaths {
			modPath, err := resolvePath(root.Path(), rawPath)
			if err != nil {
				jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
					Type:    lsp.Warning,
					Message: fmt.Sprintf("Ignoring module path %s: %s", rawPath, err),
				})
				continue
			}

			err = svc.watcher.AddModule(modPath)
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
	absPath, err := filepath.Abs(path)
	return toLowerVolumePath(absPath), err
}

func toLowerVolumePath(path string) string {
	volume := filepath.VolumeName(path)
	return strings.ToLower(volume) + path[len(volume):]
}
