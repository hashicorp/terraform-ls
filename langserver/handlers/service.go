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
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/internal/watcher"
	"github.com/hashicorp/terraform-ls/langserver/session"
	"github.com/sourcegraph/go-lsp"
)

type service struct {
	logger *log.Logger

	srvCtx context.Context

	sessCtx     context.Context
	stopSession context.CancelFunc

	watcher              watcher.Watcher
	walker               *rootmodule.Walker
	newRootModuleManager rootmodule.RootModuleManagerFactory
	newWatcher           watcher.WatcherFactory
	newWalker            rootmodule.WalkerFactory
}

var discardLogs = log.New(ioutil.Discard, "", 0)

func NewSession(srvCtx context.Context) session.Session {
	sessCtx, stopSession := context.WithCancel(srvCtx)
	return &service{
		logger:               discardLogs,
		srvCtx:               srvCtx,
		sessCtx:              sessCtx,
		stopSession:          stopSession,
		newRootModuleManager: rootmodule.NewRootModuleManager,
		newWatcher:           watcher.NewWatcher,
		newWalker:            rootmodule.NewWalker,
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

	fs := filesystem.NewFilesystem()
	fs.SetLogger(svc.logger)
	lh := LogHandler(svc.logger)
	cc := &lsp.ClientCapabilities{}

	rmm := svc.newRootModuleManager(svc.sessCtx)
	rmm.SetLogger(svc.logger)

	svc.walker = svc.newWalker()

	// The following is set via CLI flags, hence available in the server context
	if path, ok := lsctx.TerraformExecPath(svc.srvCtx); ok {
		rmm.SetTerraformExecPath(path)
	}
	if path, ok := lsctx.TerraformExecLogPath(svc.srvCtx); ok {
		rmm.SetTerraformExecLogPath(path)
	}
	if timeout, ok := lsctx.TerraformExecTimeout(svc.srvCtx); ok {
		rmm.SetTerraformExecTimeout(timeout)
	}

	ww, err := svc.newWatcher()
	if err != nil {
		return nil, err
	}
	svc.watcher = ww
	svc.watcher.SetLogger(svc.logger)
	svc.watcher.AddChangeHook(func(file watcher.TrackedFile) error {
		w, err := rmm.RootModuleByPath(file.Path())
		if err != nil {
			return err
		}
		if w.IsKnownPluginLockFile(file.Path()) {
			svc.logger.Printf("detected plugin cache change, updating ...")
			return w.UpdatePluginCache(file)
		}

		return nil
	})
	svc.watcher.AddChangeHook(func(file watcher.TrackedFile) error {
		rm, err := rmm.RootModuleByPath(file.Path())
		if err != nil {
			return err
		}
		if rm.IsKnownModuleManifestFile(file.Path()) {
			svc.logger.Printf("detected module manifest change, updating ...")
			return rm.UpdateModuleManifest(file)
		}

		return nil
	})
	err = svc.watcher.Start()
	if err != nil {
		return nil, err
	}

	rootDir := ""

	m := map[string]rpch.Func{
		"initialize": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.Initialize(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			ctx = lsctx.WithClientCapabilitiesSetter(cc, ctx)
			ctx = lsctx.WithWatcher(ww, ctx)
			ctx = lsctx.WithRootModuleWalker(svc.walker, ctx)
			ctx = lsctx.WithRootDirectory(&rootDir, ctx)
			ctx = lsctx.WithRootModuleManager(rmm, ctx)

			return handle(ctx, req, lh.Initialize)
		},
		"initialized": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.ConfirmInitialization(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)

			return handle(ctx, req, Initialized)
		},
		"textDocument/didChange": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidChange)
		},
		"textDocument/didOpen": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			ctx = lsctx.WithRootDirectory(&rootDir, ctx)
			ctx = lsctx.WithRootModuleCandidateFinder(rmm, ctx)
			return handle(ctx, req, TextDocumentDidOpen)
		},
		"textDocument/didClose": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidClose)
		},
		"textDocument/completion": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithFilesystem(fs, ctx) // TODO: Read-only FS
			ctx = lsctx.WithClientCapabilities(cc, ctx)
			ctx = lsctx.WithParserFinder(rmm, ctx)

			return handle(ctx, req, lh.TextDocumentComplete)
		},
		"textDocument/formatting": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithFilesystem(fs, ctx)
			ctx = lsctx.WithTerraformExecFinder(rmm, ctx)

			return handle(ctx, req, lh.TextDocumentFormatting)
		},
		"shutdown": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.Shutdown(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
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

	// TODO: Cancel any operations on tracked root modules
	// https://github.com/hashicorp/terraform-ls/issues/195

	if svc.walker != nil {
		svc.logger.Printf("Stopping walker for session ...")
		svc.walker.Stop()
	}

	if svc.watcher != nil {
		svc.logger.Println("Stopping watcher for session ...")
		err := svc.watcher.Stop()
		if err != nil {
			svc.logger.Println("Unable to stop watcher for session:", err)
		}
	}

	svc.stopSession()
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
