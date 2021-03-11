package handlers

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/code"
	rpch "github.com/creachadair/jrpc2/handler"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/settings"
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
	watcher          module.Watcher
	walker           *module.Walker
	modMgr           module.ModuleManager
	newModuleManager module.ModuleManagerFactory
	newWatcher       module.WatcherFactory
	newWalker        module.WalkerFactory
	tfDiscoFunc      discovery.DiscoveryFunc
	tfExecFactory    exec.ExecutorFactory
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

	svc.fs.SetLogger(svc.logger)

	lh := LogHandler(svc.logger)
	cc := &lsp.ClientCapabilities{}

	// The following is set via CLI flags, hence available in the server context
	execOpts := &exec.ExecutorOpts{}
	if path, ok := lsctx.TerraformExecPath(svc.srvCtx); ok {
		execOpts.ExecPath = path
	} else {
		tfExecPath, err := svc.tfDiscoFunc()
		if err == nil {
			execOpts.ExecPath = tfExecPath
		}
	}
	if path, ok := lsctx.TerraformExecLogPath(svc.srvCtx); ok {
		execOpts.ExecLogPath = path
	}
	if timeout, ok := lsctx.TerraformExecTimeout(svc.srvCtx); ok {
		execOpts.Timeout = timeout
	}

	svc.sessCtx = exec.WithExecutorOpts(svc.sessCtx, execOpts)
	svc.sessCtx = exec.WithExecutorFactory(svc.sessCtx, svc.tfExecFactory)

	svc.modMgr = svc.newModuleManager(svc.sessCtx, svc.fs)
	svc.modMgr.SetLogger(svc.logger)

	svc.walker = svc.newWalker(svc.fs, svc.modMgr)

	ww, err := svc.newWatcher(svc.fs, svc.modMgr)
	if err != nil {
		return nil, err
	}
	svc.watcher = ww
	svc.watcher.SetLogger(svc.logger)
	err = svc.watcher.Start()
	if err != nil {
		return nil, err
	}

	diags := diagnostics.NewNotifier(svc.sessCtx, svc.logger)

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
			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithClientCapabilitiesSetter(ctx, cc)
			ctx = lsctx.WithWatcher(ctx, ww)
			ctx = lsctx.WithModuleWalker(ctx, svc.walker)
			ctx = lsctx.WithRootDirectory(ctx, &rootDir)
			ctx = lsctx.WithCommandPrefix(ctx, &commandPrefix)
			ctx = lsctx.WithClientName(ctx, &clientName)
			ctx = lsctx.WithModuleManager(ctx, svc.modMgr)
			ctx = lsctx.WithExperimentalFeatures(ctx, &expFeatures)

			version, ok := lsctx.LanguageServerVersion(svc.srvCtx)
			if ok {
				ctx = lsctx.WithLanguageServerVersion(ctx, version)
			}

			return handle(ctx, req, lh.Initialize)
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
			ctx = lsctx.WithDiagnostics(ctx, diags)
			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithModuleManager(ctx, svc.modMgr)
			return handle(ctx, req, TextDocumentDidChange)
		},
		"textDocument/didOpen": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithDiagnostics(ctx, diags)
			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithRootDirectory(ctx, &rootDir)
			ctx = lsctx.WithModuleManager(ctx, svc.modMgr)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)
			ctx = lsctx.WithModuleWalker(ctx, svc.walker)
			ctx = exec.WithExecutorOpts(ctx, execOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)
			ctx = lsctx.WithWatcher(ctx, ww)
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
			ctx = lsctx.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)

			return handle(ctx, req, lh.TextDocumentSymbol)
		},
		"textDocument/documentLink": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithClientName(ctx, &clientName)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)

			return handle(ctx, req, lh.TextDocumentLink)
		},
		"textDocument/completion": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)

			return handle(ctx, req, lh.TextDocumentComplete)
		},
		"textDocument/hover": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithClientName(ctx, &clientName)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)

			return handle(ctx, req, lh.TextDocumentHover)
		},
		"textDocument/formatting": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)
			ctx = exec.WithExecutorOpts(ctx, execOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)

			return handle(ctx, req, lh.TextDocumentFormatting)
		},
		"textDocument/semanticTokens/full": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)

			return handle(ctx, req, lh.TextDocumentSemanticTokensFull)
		},
		"textDocument/didSave": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDiagnostics(ctx, diags)
			ctx = lsctx.WithExperimentalFeatures(ctx, &expFeatures)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)
			ctx = exec.WithExecutorOpts(ctx, execOpts)

			return handle(ctx, req, lh.TextDocumentDidSave)
		},
		"workspace/executeCommand": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithCommandPrefix(ctx, &commandPrefix)
			ctx = lsctx.WithModuleManager(ctx, svc.modMgr)
			ctx = lsctx.WithModuleWalker(ctx, svc.walker)
			ctx = lsctx.WithWatcher(ctx, ww)
			ctx = lsctx.WithRootDirectory(ctx, &rootDir)
			ctx = lsctx.WithDiagnostics(ctx, diags)
			ctx = exec.WithExecutorOpts(ctx, execOpts)
			ctx = exec.WithExecutorFactory(ctx, svc.tfExecFactory)

			return handle(ctx, req, lh.WorkspaceExecuteCommand)
		},
		"workspace/symbol": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithDocumentStorage(ctx, svc.fs)
			ctx = lsctx.WithClientCapabilities(ctx, cc)
			ctx = lsctx.WithModuleFinder(ctx, svc.modMgr)

			return handle(ctx, req, lh.WorkspaceSymbol)
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

	return convertMap(m), nil
}

func (svc *service) Finish(status jrpc2.ServerStatus) {
	if status.Closed() || status.Err != nil {
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
