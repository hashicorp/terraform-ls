// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	serverCaps := initializeResult(ctx)

	out, err := settings.DecodeOptions(params.InitializationOptions)
	if err != nil {
		return serverCaps, err
	}

	err = out.Options.Validate()
	if err != nil {
		return serverCaps, err
	}

	properties := getTelemetryProperties(out)
	properties["lsVersion"] = serverCaps.ServerInfo.Version

	clientCaps := params.Capabilities
	expClientCaps := lsp.ExperimentalClientCapabilities(clientCaps.Experimental)

	svc.server = jrpc2.ServerFromContext(ctx)

	setupTelemetry(expClientCaps, svc, ctx, properties)
	defer svc.telemetry.SendEvent(ctx, "initialize", properties)

	if params.ClientInfo.Name != "" {
		err = ilsp.SetClientName(ctx, params.ClientInfo.Name)
		if err != nil {
			return serverCaps, err
		}
	}

	expServerCaps := lsp.ExperimentalServerCapabilities{}

	if _, ok := expClientCaps.ShowReferencesCommandId(); ok {
		expServerCaps.ReferenceCountCodeLens = true
		properties["experimentalCapabilities.referenceCountCodeLens"] = true
	}
	if _, ok := expClientCaps.RefreshModuleProvidersCommandId(); ok {
		expServerCaps.RefreshModuleProviders = true
		properties["experimentalCapabilities.refreshModuleProviders"] = true
	}
	if _, ok := expClientCaps.RefreshModuleCallsCommandId(); ok {
		expServerCaps.RefreshModuleCalls = true
		properties["experimentalCapabilities.refreshModuleCalls"] = true
	}
	if _, ok := expClientCaps.RefreshTerraformVersionCommandId(); ok {
		expServerCaps.RefreshTerraformVersion = true
		properties["experimentalCapabilities.refreshTerraformVersion"] = true
	}

	serverCaps.Capabilities.Experimental = expServerCaps

	err = ilsp.SetClientCapabilities(ctx, &clientCaps)
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

	if params.RootURI == "" {
		svc.singleFileMode = true
		properties["root_uri"] = "file"
		if properties["options.ignoreSingleFileWarning"] == false {
			jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
				Type:    lsp.Warning,
				Message: "Some capabilities may be reduced when editing a single file. We recommend opening a directory for full functionality. Use 'ignoreSingleFileWarning' to suppress this warning.",
			})
		}
	} else {
		rootURI := string(params.RootURI)

		invalidUriErr := jrpc2.Errorf(jrpc2.InvalidParams,
			"Unsupported or invalid URI: %q "+
				"This is most likely client bug, please report it.", rootURI)

		if uri.IsWSLURI(rootURI) {
			properties["root_uri"] = "invalid"
			// For WSL URIs we return additional error data
			// such that clients (e.g. VS Code) can provide better UX
			// and nudge users to open in the WSL Remote Extension instead.
			return serverCaps, invalidUriErr.WithData("INVALID_URI_WSL")
		}

		if !uri.IsURIValid(rootURI) {
			properties["root_uri"] = "invalid"

			return serverCaps, invalidUriErr
		}

		err := svc.setupWalker(ctx, params, cfgOpts)
		if err != nil {
			return serverCaps, err
		}
	}

	// Walkers run asynchronously so we're intentionally *not*
	// passing the request context here
	// Static user-provided paths take precedence over dynamic discovery
	walkerCtx := context.Background()
	walkerCtx = lsctx.WithRPCContext(walkerCtx, lsctx.RPCContext(ctx))

	err = svc.closedDirWalker.StartWalking(walkerCtx)
	if err != nil {
		return serverCaps, fmt.Errorf("failed to start closedDirWalker: %w", err)
	}
	err = svc.openDirWalker.StartWalking(walkerCtx)
	if err != nil {
		return serverCaps, fmt.Errorf("failed to start openDirWalker: %w", err)
	}

	return serverCaps, err
}

func setupTelemetry(expClientCaps lsp.ExpClientCapabilities, svc *service, ctx context.Context, properties map[string]interface{}) {
	if tv, ok := expClientCaps.TelemetryVersion(); ok {
		svc.logger.Printf("enabling telemetry (version: %d)", tv)
		err := svc.setupTelemetry(tv, svc.server)
		if err != nil {
			svc.logger.Printf("failed to setup telemetry: %s", err)
		}
		svc.logger.Printf("telemetry enabled (version: %d)", tv)
	}
}

func getTelemetryProperties(out *settings.DecodedOptions) map[string]interface{} {
	properties := map[string]interface{}{
		"experimentalCapabilities.referenceCountCodeLens": false,
		"options.ignoreSingleFileWarning":                 false,
		"options.rootModulePaths":                         false,
		"options.excludeModulePaths":                      false,
		"options.commandPrefix":                           false,
		"options.indexing.ignoreDirectoryNames":           false,
		"options.indexing.ignorePaths":                    false,
		"options.experimentalFeatures.validateOnSave":     false,
		"options.terraform.path":                          false,
		"options.terraform.timeout":                       "",
		"options.terraform.logFilePath":                   false,
		"root_uri":                                        "dir",
		"lsVersion":                                       "",
	}

	properties["options.rootModulePaths"] = len(out.Options.XLegacyModulePaths) > 0
	properties["options.excludeModulePaths"] = len(out.Options.XLegacyExcludeModulePaths) > 0
	properties["options.commandPrefix"] = len(out.Options.CommandPrefix) > 0
	properties["options.indexing.ignoreDirectoryNames"] = len(out.Options.Indexing.IgnoreDirectoryNames) > 0
	properties["options.indexing.ignorePaths"] = len(out.Options.Indexing.IgnorePaths) > 0
	properties["options.experimentalFeatures.prefillRequiredFields"] = out.Options.ExperimentalFeatures.PrefillRequiredFields
	properties["options.experimentalFeatures.validateOnSave"] = out.Options.ExperimentalFeatures.ValidateOnSave
	properties["options.ignoreSingleFileWarning"] = out.Options.IgnoreSingleFileWarning
	properties["options.terraform.path"] = len(out.Options.Terraform.Path) > 0
	properties["options.terraform.timeout"] = out.Options.Terraform.Timeout
	properties["options.terraform.logFilePath"] = len(out.Options.Terraform.LogFilePath) > 0

	return properties
}

func initializeResult(ctx context.Context) lsp.InitializeResult {
	serverCaps := lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: lsp.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    lsp.Incremental,
			},
			CompletionProvider: lsp.CompletionOptions{
				ResolveProvider:   true,
				TriggerCharacters: []string{".", "["},
			},
			CodeActionProvider: lsp.CodeActionOptions{
				CodeActionKinds: ilsp.SupportedCodeActions.AsSlice(),
				ResolveProvider: false,
			},
			DeclarationProvider:        true,
			DefinitionProvider:         true,
			CodeLensProvider:           &lsp.CodeLensOptions{},
			ReferencesProvider:         true,
			HoverProvider:              true,
			DocumentFormattingProvider: true,
			DocumentSymbolProvider:     true,
			WorkspaceSymbolProvider:    true,
			Workspace: lsp.Workspace6Gn{
				WorkspaceFolders: lsp.WorkspaceFolders5Gn{
					Supported:           true,
					ChangeNotifications: "workspace/didChangeWorkspaceFolders",
				},
			},
			SignatureHelpProvider: lsp.SignatureHelpOptions{
				TriggerCharacters: []string{"(", ","},
			},
		},
	}

	serverCaps.ServerInfo.Name = "terraform-ls"
	version, ok := lsctx.LanguageServerVersion(ctx)
	if ok {
		serverCaps.ServerInfo.Version = version
	}

	return serverCaps
}

func (svc *service) setupWalker(ctx context.Context, params lsp.InitializeParams, options *settings.Options) error {
	rootURI := string(params.RootURI)
	root := document.DirHandleFromURI(rootURI)

	err := lsctx.SetRootDirectory(ctx, root.Path())
	if err != nil {
		return err
	}

	if len(options.XLegacyModulePaths) != 0 {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("rootModulePaths (%q) is deprecated (no-op), add a folder to workspace "+
				"instead if you'd like it to be indexed", options.XLegacyModulePaths),
		})
	}
	if len(options.XLegacyExcludeModulePaths) != 0 {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("excludeModulePaths (%q) is deprecated (no-op), use indexing.ignorePaths instead",
				options.XLegacyExcludeModulePaths),
		})
	}
	if len(options.XLegacyIgnoreDirectoryNames) != 0 {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("ignoreDirectoryNames (%q) is deprecated (no-op), use indexing.ignoreDirectoryNames instead",
				options.XLegacyIgnoreDirectoryNames),
		})
	}

	var ignoredPaths []string
	for _, rawPath := range options.Indexing.IgnorePaths {
		modPath, err := resolvePath(root.Path(), rawPath)
		if err != nil {
			jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
				Type: lsp.Warning,
				Message: fmt.Sprintf("Unable to ignore path (unsupported or invalid URI): %s: %s",
					rawPath, err),
			})
			continue
		}
		ignoredPaths = append(ignoredPaths, modPath)
	}

	err = svc.stateStore.WalkerPaths.EnqueueDir(ctx, root)
	if err != nil {
		return err
	}

	if len(params.WorkspaceFolders) > 0 {
		for _, folder := range params.WorkspaceFolders {
			if !uri.IsURIValid(folder.URI) {
				jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
					Type: lsp.Warning,
					Message: fmt.Sprintf("Ignoring workspace folder (unsupported or invalid URI) %s."+
						" This is most likely bug, please report it.", folder.URI),
				})
				continue
			}

			modPath := document.DirHandleFromURI(folder.URI)

			err := svc.stateStore.WalkerPaths.EnqueueDir(ctx, modPath)
			if err != nil {
				jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
					Type: lsp.Warning,
					Message: fmt.Sprintf("Ignoring workspace folder %s: %s."+
						" This is most likely bug, please report it.", folder.URI, err),
				})
				continue
			}
		}
	}

	svc.closedDirWalker.SetIgnoredDirectoryNames(options.Indexing.IgnoreDirectoryNames)
	svc.closedDirWalker.SetIgnoredPaths(ignoredPaths)
	svc.openDirWalker.SetIgnoredDirectoryNames(options.Indexing.IgnoreDirectoryNames)
	svc.openDirWalker.SetIgnoredPaths(ignoredPaths)

	return nil
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
