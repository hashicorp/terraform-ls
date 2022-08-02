package handlers

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/code"
	rpch "github.com/creachadair/jrpc2/handler"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/indexer"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/langserver/notifier"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/scheduler"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	"github.com/hashicorp/terraform-ls/internal/settings"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/telemetry"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
)

type service struct {
	logger *log.Logger

	srvCtx context.Context

	sessCtx     context.Context
	stopSession context.CancelFunc

	lowPrioIndexer  *scheduler.Scheduler
	highPrioIndexer *scheduler.Scheduler

	closedDirWalker *walker.Walker
	openDirWalker   *walker.Walker

	fs               *filesystem.Filesystem
	modStore         *state.ModuleStore
	schemaStore      *state.ProviderSchemaStore
	regMetadataStore *state.RegistryModuleStore
	tfDiscoFunc      discovery.DiscoveryFunc
	tfExecFactory    exec.ExecutorFactory
	tfExecOpts       *exec.ExecutorOpts
	telemetry        telemetry.Sender
	decoder          *decoder.Decoder
	stateStore       *state.StateStore
	server           session.Server
	diagsNotifier    *diagnostics.Notifier
	notifier         *notifier.Notifier
	indexer          *indexer.Indexer
	registryClient   registry.Client

	walkerCollector    *walker.WalkerCollector
	additionalHandlers map[string]rpch.Func

	singleFileMode bool
}

var discardLogs = log.New(ioutil.Discard, "", 0)

func NewSession(srvCtx context.Context) session.Session {
	d := &discovery.Discovery{}

	sessCtx, stopSession := context.WithCancel(srvCtx)
	return &service{
		logger:         discardLogs,
		srvCtx:         srvCtx,
		sessCtx:        sessCtx,
		stopSession:    stopSession,
		tfDiscoFunc:    d.LookPath,
		tfExecFactory:  exec.NewExecutor,
		telemetry:      &telemetry.NoopSender{},
		registryClient: registry.NewClient(),
	}
}

func (svc *service) SetLogger(logger *log.Logger) {
	svc.logger = logger
}

// Assigner builds out the jrpc2.Map according to the LSP protocol
// and passes related dependencies to handlers via context
func (svc *service) Assigner() (jrpc2.Assigner, error) {
	svc.logger.Println("Preparing new session ...")

	session := session.NewSession(svc.stopSession)

	err := session.Prepare()
	if err != nil {
		return nil, fmt.Errorf("Unable to prepare session: %w", err)
	}

	svc.telemetry = &telemetry.NoopSender{Logger: svc.logger}

	cc := &lsp.ClientCapabilities{}

	rootDir := ""
	commandPrefix := ""
	clientName := ""
	var expFeatures settings.ExperimentalFeatures

	m := map[string]rpch.Func{
		"initialize": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.Initialize(req)
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithRootDirectory(ctx, &rootDir)
			ctx = lsctx.WithCommandPrefix(ctx, &commandPrefix)
			ctx = ilsp.ContextWithClientName(ctx, &clientName)
			ctx = lsctx.WithExperimentalFeatures(ctx, &expFeatures)

			version, ok := lsctx.LanguageServerVersion(svc.srvCtx)
			if ok {
				ctx = lsctx.WithLanguageServerVersion(ctx, version)
			}

			return handle(ctx, req, svc.Initialize)
		},
		"initialized": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.ConfirmInitialization(req)
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.Initialized)
		},
		"textDocument/didChange": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			return handle(ctx, req, svc.TextDocumentDidChange)
		},
		"textDocument/didOpen": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			return handle(ctx, req, svc.TextDocumentDidOpen)
		},
		"textDocument/didClose": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			return handle(ctx, req, svc.TextDocumentDidClose)
		},
		"textDocument/documentSymbol": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.TextDocumentSymbol)
		},
		"textDocument/documentLink": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)
			ctx = ilsp.ContextWithClientName(ctx, &clientName)

			return handle(ctx, req, svc.TextDocumentLink)
		},
		"textDocument/declaration": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.GoToDeclaration)
		},
		"textDocument/definition": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.GoToDefinition)
		},
		"textDocument/completion": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithExperimentalFeatures(ctx, &expFeatures)

			return handle(ctx, req, svc.TextDocumentComplete)
		},
		"completionItem/resolve": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithExperimentalFeatures(ctx, &expFeatures)

			return handle(ctx, req, svc.CompletionItemResolve)
		},
		"textDocument/hover": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)
			ctx = ilsp.ContextWithClientName(ctx, &clientName)

			return handle(ctx, req, svc.TextDocumentHover)
		},
		"textDocument/codeAction": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)
			ctx = exec.WithExecutorOpts(ctx, svc.tfExecOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)

			return handle(ctx, req, svc.TextDocumentCodeAction)
		},
		"textDocument/codeLens": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.TextDocumentCodeLens)
		},
		"textDocument/formatting": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = exec.WithExecutorOpts(ctx, svc.tfExecOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)

			return handle(ctx, req, svc.TextDocumentFormatting)
		},
		"textDocument/semanticTokens/full": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.TextDocumentSemanticTokensFull)
		},
		"textDocument/didSave": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDiagnosticsNotifier(ctx, svc.diagsNotifier)
			ctx = lsctx.WithExperimentalFeatures(ctx, &expFeatures)
			ctx = exec.WithExecutorOpts(ctx, svc.tfExecOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)

			return handle(ctx, req, svc.TextDocumentDidSave)
		},
		"workspace/didChangeWorkspaceFolders": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			return handle(ctx, req, svc.DidChangeWorkspaceFolders)
		},
		"workspace/didChangeWatchedFiles": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			return handle(ctx, req, svc.DidChangeWatchedFiles)
		},
		"textDocument/references": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			return handle(ctx, req, svc.References)
		},
		"workspace/executeCommand": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithCommandPrefix(ctx, &commandPrefix)
			ctx = lsctx.WithRootDirectory(ctx, &rootDir)
			ctx = lsctx.WithDiagnosticsNotifier(ctx, svc.diagsNotifier)
			ctx = ilsp.ContextWithClientName(ctx, &clientName)
			ctx = exec.WithExecutorOpts(ctx, svc.tfExecOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)

			return handle(ctx, req, svc.WorkspaceExecuteCommand)
		},
		"workspace/symbol": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.WorkspaceSymbol)
		},
		"shutdown": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.Shutdown(req)
			if err != nil {
				return nil, err
			}
			svc.shutdown()
			return handle(ctx, req, Shutdown)
		},
		"exit": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.Exit()
			if err != nil {
				return nil, err
			}

			svc.stopSession()

			return nil, nil
		},
		"$/cancelRequest": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			return handle(ctx, req, CancelRequest)
		},
	}

	// For use in tests, e.g. to test request cancellation
	if len(svc.additionalHandlers) > 0 {
		for methodName, handlerFunc := range svc.additionalHandlers {
			m[methodName] = handlerFunc
		}
	}

	return convertMap(m), nil
}

func (svc *service) configureSessionDependencies(ctx context.Context, cfgOpts *settings.Options) error {
	// Raise warnings for deprecated options
	if cfgOpts.XLegacyTerraformExecPath != "" {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("terraformExecPath (%q) is deprecated (no-op), use terraform.path instead",
				cfgOpts.XLegacyExcludeModulePaths),
		})
	}
	if cfgOpts.XLegacyTerraformExecTimeout != "" {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("terraformExecTimeout (%q) is deprecated (no-op), use terraform.timeout instead",
				cfgOpts.XLegacyExcludeModulePaths),
		})
	}
	if cfgOpts.XLegacyTerraformExecLogFilePath != "" {
		jrpc2.ServerFromContext(ctx).Notify(ctx, "window/showMessage", &lsp.ShowMessageParams{
			Type: lsp.Warning,
			Message: fmt.Sprintf("terraformExecLogFilePath (%q) is deprecated (no-op), use terraform.logFilePath instead",
				cfgOpts.XLegacyExcludeModulePaths),
		})
	}

	// The following is set via CLI flags, hence available in the server context
	execOpts := &exec.ExecutorOpts{}
	if len(cfgOpts.Terraform.Path) > 0 {
		execOpts.ExecPath = cfgOpts.Terraform.Path
	} else {
		path, err := svc.tfDiscoFunc()
		if err == nil {
			execOpts.ExecPath = path
		}
	}
	svc.srvCtx = lsctx.WithTerraformExecPath(svc.srvCtx, execOpts.ExecPath)

	if len(cfgOpts.Terraform.LogFilePath) > 0 {
		execOpts.ExecLogPath = cfgOpts.Terraform.LogFilePath
	}

	if len(cfgOpts.Terraform.Timeout) > 0 {
		d, err := time.ParseDuration(cfgOpts.Terraform.Timeout)
		if err != nil {
			return fmt.Errorf("Failed to parse terraform.timeout LSP config option: %s", err)
		}
		execOpts.Timeout = d
	}

	svc.diagsNotifier = diagnostics.NewNotifier(svc.server, svc.logger)

	svc.tfExecOpts = execOpts

	svc.sessCtx = exec.WithExecutorOpts(svc.sessCtx, execOpts)
	svc.sessCtx = exec.WithExecutorFactory(svc.sessCtx, svc.tfExecFactory)

	if svc.stateStore == nil {
		store, err := state.NewStateStore()
		if err != nil {
			return err
		}
		svc.stateStore = store
	}

	svc.stateStore.SetLogger(svc.logger)

	moduleHooks := []notifier.Hook{
		updateDiagnostics(svc.diagsNotifier),
		sendModuleTelemetry(svc.stateStore, svc.telemetry),
	}

	svc.lowPrioIndexer = scheduler.NewScheduler(svc.stateStore.JobStore, 1, job.LowPriority)
	svc.lowPrioIndexer.SetLogger(svc.logger)
	svc.lowPrioIndexer.Start(svc.sessCtx)
	svc.logger.Printf("started low priority scheduler")

	svc.highPrioIndexer = scheduler.NewScheduler(svc.stateStore.JobStore, 1, job.HighPriority)
	svc.highPrioIndexer.SetLogger(svc.logger)
	svc.highPrioIndexer.Start(svc.sessCtx)
	svc.logger.Printf("started high priority scheduler")

	cc, err := ilsp.ClientCapabilities(ctx)
	if err == nil {
		if _, ok := lsp.ExperimentalClientCapabilities(cc.Experimental).ShowReferencesCommandId(); ok {
			moduleHooks = append(moduleHooks, refreshCodeLens(svc.server))
		}

		if commandId, ok := lsp.ExperimentalClientCapabilities(cc.Experimental).RefreshModuleProvidersCommandId(); ok {
			moduleHooks = append(moduleHooks, callRefreshClientCommand(svc.server, commandId))
		}

		if commandId, ok := lsp.ExperimentalClientCapabilities(cc.Experimental).RefreshModuleCallsCommandId(); ok {
			moduleHooks = append(moduleHooks, callRefreshClientCommand(svc.server, commandId))
		}

		if commandId, ok := lsp.ExperimentalClientCapabilities(cc.Experimental).RefreshTerraformVersionCommandId(); ok {
			moduleHooks = append(moduleHooks, callRefreshClientCommand(svc.server, commandId))
		}

		if cc.Workspace.SemanticTokens.RefreshSupport {
			moduleHooks = append(moduleHooks, refreshSemanticTokens(svc.server))
		}
	}

	svc.notifier = notifier.NewNotifier(svc.stateStore.Modules, moduleHooks)
	svc.notifier.SetLogger(svc.logger)
	svc.notifier.Start(svc.sessCtx)

	svc.modStore = svc.stateStore.Modules
	svc.schemaStore = svc.stateStore.ProviderSchemas

	svc.fs = filesystem.NewFilesystem(svc.stateStore.DocumentStore)
	svc.fs.SetLogger(svc.logger)

	svc.indexer = indexer.NewIndexer(svc.fs, svc.modStore, svc.schemaStore, svc.stateStore.RegistryModules,
		svc.stateStore.JobStore, svc.tfExecFactory, svc.registryClient)
	svc.indexer.SetLogger(svc.logger)

	svc.decoder = decoder.NewDecoder(&idecoder.PathReader{
		ModuleReader: svc.modStore,
		SchemaReader: svc.schemaStore,
	})
	decoderContext := idecoder.DecoderContext(ctx)
	svc.AppendCompletionHooks(decoderContext)
	svc.decoder.SetContext(decoderContext)

	err = schemas.PreloadSchemasToStore(svc.stateStore.ProviderSchemas)
	if err != nil {
		return err
	}

	closedPa := state.NewPathAwaiter(svc.stateStore.WalkerPaths, false)
	svc.closedDirWalker = walker.NewWalker(svc.fs, closedPa, svc.modStore, svc.indexer.WalkedModule)
	svc.closedDirWalker.Collector = svc.walkerCollector
	svc.closedDirWalker.SetLogger(svc.logger)

	opendPa := state.NewPathAwaiter(svc.stateStore.WalkerPaths, true)
	svc.openDirWalker = walker.NewWalker(svc.fs, opendPa, svc.modStore, svc.indexer.WalkedModule)
	svc.closedDirWalker.Collector = svc.walkerCollector
	svc.openDirWalker.SetLogger(svc.logger)

	return nil
}

func (svc *service) setupTelemetry(version int, notifier session.ClientNotifier) error {
	t, err := telemetry.NewSender(version, notifier)
	if err != nil {
		return err
	}

	svc.telemetry = t
	return nil
}

func (svc *service) Finish(_ jrpc2.Assigner, status jrpc2.ServerStatus) {
	if status.Closed || status.Err != nil {
		svc.logger.Printf("session stopped unexpectedly (err: %v)", status.Err)
	}

	svc.shutdown()
	svc.stopSession()
}

func (svc *service) shutdown() {
	if svc.closedDirWalker != nil {
		svc.logger.Printf("stopping closedDirWalker for session ...")
		svc.closedDirWalker.Stop()
		svc.logger.Printf("closedDirWalker stopped")
	}
	if svc.openDirWalker != nil {
		svc.logger.Printf("stopping openDirWalker for session ...")
		svc.openDirWalker.Stop()
		svc.logger.Printf("openDirWalker stopped")
	}

	if svc.lowPrioIndexer != nil {
		svc.lowPrioIndexer.Stop()
	}
	if svc.highPrioIndexer != nil {
		svc.highPrioIndexer.Stop()
	}
}

// convertMap is a helper function allowing us to omit the jrpc2.Func
// signature from the method definitions
func convertMap(m map[string]rpch.Func) rpch.Map {
	hm := make(rpch.Map, len(m))

	for method, fun := range m {
		hm[method] = rpch.New(fun)
	}

	return hm
}

const requestCancelled code.Code = -32800

// handle calls a jrpc2.Func compatible function
func handle(ctx context.Context, req *jrpc2.Request, fn interface{}) (interface{}, error) {
	f := rpch.New(fn)
	result, err := f.Handle(ctx, req)
	if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
		err = fmt.Errorf("%w: %s", requestCancelled.Err(), err)
	}
	return result, err
}

func (svc *service) decoderForDocument(ctx context.Context, doc *document.Document) (*decoder.PathDecoder, error) {
	return svc.decoder.Path(lang.Path{
		Path:       doc.Dir.Path(),
		LanguageID: doc.LanguageID,
	})
}
