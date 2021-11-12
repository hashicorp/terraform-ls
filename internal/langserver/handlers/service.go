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
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	"github.com/hashicorp/terraform-ls/internal/settings"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/telemetry"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

type service struct {
	logger *log.Logger

	srvCtx context.Context

	sessCtx     context.Context
	stopSession context.CancelFunc

	fs               filesystem.Filesystem
	modStore         *state.ModuleStore
	schemaStore      *state.ProviderSchemaStore
	watcher          module.Watcher
	walker           *module.Walker
	modMgr           module.ModuleManager
	newModuleManager module.ModuleManagerFactory
	newWatcher       module.WatcherFactory
	newWalker        module.WalkerFactory
	tfDiscoFunc      discovery.DiscoveryFunc
	tfExecFactory    exec.ExecutorFactory
	tfExecOpts       *exec.ExecutorOpts
	telemetry        telemetry.Sender
	decoder          *decoder.Decoder
	stateStore       *state.StateStore
	server           session.Server
	diagsNotifier    *diagnostics.Notifier

	additionalHandlers map[string]rpch.Func
}

var discardLogs = log.New(ioutil.Discard, "", 0)

func NewSession(srvCtx context.Context) session.Session {
	fs := filesystem.NewFilesystem()
	d := &discovery.Discovery{}

	sessCtx, stopSession := context.WithCancel(srvCtx)
	return &service{
		logger:           discardLogs,
		fs:               fs,
		srvCtx:           srvCtx,
		sessCtx:          sessCtx,
		stopSession:      stopSession,
		newModuleManager: module.NewModuleManager,
		newWatcher:       module.NewWatcher,
		newWalker:        module.NewWalker,
		tfDiscoFunc:      d.LookPath,
		tfExecFactory:    exec.NewExecutor,
		telemetry:        &telemetry.NoopSender{},
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
	svc.fs.SetLogger(svc.logger)

	lh := LogHandler(svc.logger)
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

			return handle(ctx, req, Initialized)
		},
		"textDocument/didChange": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithModuleManager(ctx, svc.modMgr)
			return handle(ctx, req, TextDocumentDidChange)
		},
		"textDocument/didOpen": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithModuleManager(ctx, svc.modMgr)
			ctx = lsctx.WithWatcher(ctx, svc.watcher)
			return handle(ctx, req, lh.TextDocumentDidOpen)
		},
		"textDocument/didClose": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			return handle(ctx, req, TextDocumentDidClose)
		},
		"textDocument/documentSymbol": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.TextDocumentSymbol)
		},
		"textDocument/documentLink": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = ilsp.WithClientCapabilities(ctx, cc)
			ctx = ilsp.ContextWithClientName(ctx, &clientName)

			return handle(ctx, req, svc.TextDocumentLink)
		},
		"textDocument/declaration": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.GoToReferenceTarget)
		},
		"textDocument/definition": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.GoToReferenceTarget)
		},
		"textDocument/completion": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = ilsp.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithExperimentalFeatures(ctx, &expFeatures)

			return handle(ctx, req, svc.TextDocumentComplete)
		},
		"textDocument/hover": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
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
			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = exec.WithExecutorOpts(ctx, svc.tfExecOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)

			return handle(ctx, req, lh.TextDocumentCodeAction)
		},
		"textDocument/codeLens": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = ilsp.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)

			return handle(ctx, req, svc.TextDocumentCodeLens)
		},
		"textDocument/formatting": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = exec.WithExecutorOpts(ctx, svc.tfExecOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)

			return handle(ctx, req, lh.TextDocumentFormatting)
		},
		"textDocument/semanticTokens/full": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
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
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)
			ctx = exec.WithExecutorOpts(ctx, svc.tfExecOpts)

			return handle(ctx, req, lh.TextDocumentDidSave)
		},
		"workspace/didChangeWorkspaceFolders": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithModuleWalker(ctx, svc.walker)
			ctx = lsctx.WithModuleManager(ctx, svc.modMgr)
			ctx = lsctx.WithWatcher(ctx, svc.watcher)

			return handle(ctx, req, lh.DidChangeWorkspaceFolders)
		},
		"textDocument/references": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)

			return handle(ctx, req, svc.References)
		},
		"workspace/executeCommand": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithCommandPrefix(ctx, &commandPrefix)
			ctx = lsctx.WithModuleManager(ctx, svc.modMgr)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)
			ctx = lsctx.WithModuleWalker(ctx, svc.walker)
			ctx = lsctx.WithWatcher(ctx, svc.watcher)
			ctx = lsctx.WithRootDirectory(ctx, &rootDir)
			ctx = lsctx.WithDiagnosticsNotifier(ctx, svc.diagsNotifier)
			ctx = exec.WithExecutorOpts(ctx, svc.tfExecOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)

			return handle(ctx, req, lh.WorkspaceExecuteCommand)
		},
		"workspace/symbol": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = ilsp.WithClientCapabilities(ctx, cc)

			return handle(ctx, req, svc.WorkspaceSymbol)
		},
		"shutdown": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.Shutdown(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
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
	// The following is set via CLI flags, hence available in the server context
	execOpts := &exec.ExecutorOpts{}
	cliExecPath, ok := lsctx.TerraformExecPath(svc.srvCtx)
	if ok {
		if len(cfgOpts.TerraformExecPath) > 0 {
			return fmt.Errorf("Terraform exec path can either be set via (-tf-exec) CLI flag " +
				"or (terraformExecPath) LSP config option, not both")
		}
		execOpts.ExecPath = cliExecPath
	} else if len(cfgOpts.TerraformExecPath) > 0 {
		execOpts.ExecPath = cfgOpts.TerraformExecPath
	} else {
		path, err := svc.tfDiscoFunc()
		if err == nil {
			execOpts.ExecPath = path
		}
	}
	svc.srvCtx = lsctx.WithTerraformExecPath(svc.srvCtx, execOpts.ExecPath)

	path, ok := lsctx.TerraformExecLogPath(svc.srvCtx)
	if ok {
		if len(cfgOpts.TerraformLogFilePath) > 0 {
			return fmt.Errorf("Terraform log file path can either be set via (-tf-log-file) CLI flag " +
				"or (terraformLogFilePath) LSP config option, not both")
		}
		execOpts.ExecLogPath = path
	} else if len(cfgOpts.TerraformLogFilePath) > 0 {
		execOpts.ExecLogPath = cfgOpts.TerraformLogFilePath
	}

	timeout, ok := lsctx.TerraformExecTimeout(svc.srvCtx)
	if ok {
		if len(cfgOpts.TerraformExecTimeout) > 0 {
			return fmt.Errorf("Terraform exec timeout can either be set via (-tf-exec-timeout) CLI flag " +
				"or (terraformExecTimeout) LSP config option, not both")
		}
		execOpts.Timeout = timeout
	} else if len(cfgOpts.TerraformExecTimeout) > 0 {
		d, err := time.ParseDuration(cfgOpts.TerraformExecTimeout)
		if err != nil {
			return fmt.Errorf("Failed to parse terraformExecTimeout LSP config option: %s", err)
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
	svc.stateStore.Modules.ChangeHooks = state.ModuleChangeHooks{
		updateDiagnostics(svc.sessCtx, svc.diagsNotifier),
		sendModuleTelemetry(svc.sessCtx, svc.stateStore, svc.telemetry),
		refreshCodeLens(svc.sessCtx, svc.server),
	}

	svc.modStore = svc.stateStore.Modules
	svc.schemaStore = svc.stateStore.ProviderSchemas

	svc.decoder = idecoder.NewDecoder(ctx, &idecoder.PathReader{
		ModuleReader: svc.modStore,
		SchemaReader: svc.schemaStore,
	})

	err := schemas.PreloadSchemasToStore(svc.stateStore.ProviderSchemas)
	if err != nil {
		return err
	}

	svc.modMgr = svc.newModuleManager(svc.sessCtx, svc.fs, svc.stateStore.Modules, svc.stateStore.ProviderSchemas)
	svc.modMgr.SetLogger(svc.logger)

	svc.walker = svc.newWalker(svc.fs, svc.modMgr)
	svc.walker.SetLogger(svc.logger)

	ww, err := svc.newWatcher(svc.fs, svc.modMgr)
	if err != nil {
		return err
	}
	svc.watcher = ww
	svc.watcher.SetLogger(svc.logger)
	err = svc.watcher.Start()
	if err != nil {
		return err
	}

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
	if svc.walker != nil {
		svc.logger.Printf("stopping walker for session ...")
		svc.walker.Stop()
		svc.logger.Printf("walker stopped")
	}

	if svc.watcher != nil {
		svc.logger.Println("stopping watcher for session ...")
		err := svc.watcher.Stop()
		if err != nil {
			svc.logger.Println("unable to stop watcher for session:", err)
		} else {
			svc.logger.Println("watcher stopped")
		}
	}

	if svc.modMgr != nil {
		svc.logger.Println("cancelling any module loading ...")
		svc.modMgr.CancelLoading()
		svc.logger.Println("module loading cancelled")
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

func (svc *service) decoderForDocument(ctx context.Context, doc filesystem.Document) (*decoder.PathDecoder, error) {
	return svc.decoder.Path(lang.Path{
		Path:       doc.Dir(),
		LanguageID: doc.LanguageID(),
	})
}
