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
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/langserver/srvctl"
	"github.com/sourcegraph/go-lsp"
)

type handlerProvider struct {
	logger *log.Logger
	srvCtl srvctl.ServerController

	executorFunc func(ctx context.Context, path string) *exec.Executor
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

func New() *handlerProvider {
	return &handlerProvider{
		logger:       defaultLogger,
		executorFunc: exec.NewExecutor,
	}
}

func NewMock(mock *exec.Mock) *handlerProvider {
	return &handlerProvider{
		logger: defaultLogger,
		executorFunc: func(ctx context.Context, path string) *exec.Executor {
			return exec.MockExecutor(mock)
		},
	}
}

func (hp *handlerProvider) SetLogger(logger *log.Logger) {
	hp.logger = logger
}

// Handlers builds out the jrpc2.Map according to the LSP protocol
// and passes related dependencies to handlers via context
func (hp *handlerProvider) Handlers(ctl srvctl.ServerController) jrpc2.Assigner {
	hp.srvCtl = ctl
	fs := filesystem.NewFilesystem()
	fs.SetLogger(hp.logger)
	lh := LogHandler(hp.logger)
	cc := &lsp.ClientCapabilities{}

	m := map[string]rpch.Func{
		"initialize": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hp.srvCtl.Initialize(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			ctx = lsctx.WithClientCapabilitiesSetter(cc, ctx)

			return handle(ctx, req, Initialize)
		},
		"initialized": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hp.srvCtl.ConfirmInitialization(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)

			return handle(ctx, req, Initialized)
		},
		"textDocument/didChange": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hp.srvCtl.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidChange)
		},
		"textDocument/didOpen": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hp.srvCtl.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidOpen)
		},
		"textDocument/didClose": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hp.srvCtl.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidClose)
		},
		"textDocument/completion": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hp.srvCtl.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithFilesystem(fs, ctx) // TODO: Read-only FS
			ctx = lsctx.WithClientCapabilities(cc, ctx)

			tfPath, err := discovery.LookPath()
			if err != nil {
				return nil, err
			}
			tf := hp.executorFunc(ctx, tfPath)
			tf.SetLogger(hp.logger)
			ctx = lsctx.WithTerraformExecutor(tf, ctx)

			return handle(ctx, req, lh.TextDocumentComplete)
		},
		"shutdown": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hp.srvCtl.Shutdown(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			// TODO: Exit the process after a timeout if `exit` method is not called
			// to prevent zombie processes (?)
			return handle(ctx, req, Shutdown)
		},
		"exit": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hp.srvCtl.Exit()
			if err != nil {
				return nil, err
			}

			return handle(ctx, req, Shutdown)
		},
		"$/cancelRequest": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hp.srvCtl.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			return handle(ctx, req, CancelRequest)
		},
	}

	return convertMap(m)
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
