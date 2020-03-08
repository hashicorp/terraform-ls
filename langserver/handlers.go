package langserver

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/code"
	rpch "github.com/creachadair/jrpc2/handler"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/langserver/handlers"
	"github.com/sourcegraph/go-lsp"
)

type handlerMap struct {
	logger *log.Logger

	srv         *server
	srvStopFunc context.CancelFunc
}

// Map builds out the jrpc2.Map according to the LSP protocol
// and passes related dependencies to methods via context
func (hm *handlerMap) Map() rpch.Map {
	fs := filesystem.NewFilesystem()
	fs.SetLogger(hm.logger)
	lh := handlers.LogHandler(hm.logger)
	cc := &lsp.ClientCapabilities{}

	m := map[string]rpch.Func{
		"initialize": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hm.srv.Initialize(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			ctx = lsctx.WithClientCapabilitiesSetter(cc, ctx)

			return handle(ctx, req, handlers.Initialize)
		},
		"initialized": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hm.srv.ConfirmInitialization(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)

			return handle(ctx, req, handlers.Initialized)
		},
		"textDocument/didChange": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, handlers.TextDocumentDidChange)
		},
		"textDocument/didOpen": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, handlers.TextDocumentDidOpen)
		},
		"textDocument/didClose": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, handlers.TextDocumentDidClose)
		},
		"textDocument/completion": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
			}

			ctx = lsctx.WithFilesystem(fs, ctx) // TODO: Read-only FS
			ctx = lsctx.WithClientCapabilities(cc, ctx)

			tf := exec.NewExecutor(ctx)
			tf.SetLogger(hm.logger)
			ctx = lsctx.WithTerraformExecutor(tf, ctx)

			return handle(ctx, req, lh.TextDocumentComplete)
		},
		"shutdown": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hm.srv.Shutdown(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			// TODO: Exit the process after a timeout if `exit` method is not called
			// to prevent zombie processes (?)
			return handle(ctx, req, handlers.Shutdown)
		},
		"exit": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsDown() && !hm.srv.IsPrepared() {
				return nil, fmt.Errorf("Cannot exit as server is %s", hm.srv.State())
			}

			hm.srvStopFunc()

			return handle(ctx, req, handlers.Shutdown)
		},
		"$/cancelRequest": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
			}

			return handle(ctx, req, handlers.CancelRequest)
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
