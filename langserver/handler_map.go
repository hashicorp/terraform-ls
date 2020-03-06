package langserver

import (
	"context"
	"fmt"
	"log"

	"github.com/creachadair/jrpc2"
	rpch "github.com/creachadair/jrpc2/handler"
	lsctx "github.com/radeksimko/terraform-ls/internal/context"
	"github.com/radeksimko/terraform-ls/internal/filesystem"
	"github.com/radeksimko/terraform-ls/internal/terraform/exec"
)

// logHandler provides handlers logger
type logHandler struct {
	logger *log.Logger
}

type handlerMap struct {
	logger *log.Logger

	srv         *server
	srvStopFunc context.CancelFunc
}

// Map builds out the jrpc2.Map according to the LSP protocol
// and passes related dependencies to methods via context
func (hm *handlerMap) Map() rpch.Map {
	fs := filesystem.NewFilesystem()
	lh := logHandler{hm.logger}

	m := map[string]rpch.Func{
		"initialize": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hm.srv.Initialize(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)

			return handle(ctx, req, Initialize)
		},
		"initialized": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := hm.srv.ConfirmInitialization(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)

			return handle(ctx, req, Initialized)
		},
		"textDocument/didChange": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidChange)
		},
		"textDocument/didOpen": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidOpen)
		},
		"textDocument/didClose": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidClose)
		},
		"textDocument/completion": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
			}

			ctx = lsctx.WithFilesystem(fs, ctx) // TODO: Read-only FS

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
			return handle(ctx, req, Shutdown)
		},
		"exit": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsDown() {
				return nil, fmt.Errorf("Cannot exit as server is %s", hm.srv.State())
			}

			hm.srvStopFunc()

			return handle(ctx, req, Shutdown)
		},
		"$/cancelRequest": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			if !hm.srv.IsInitializationConfirmed() {
				return nil, SrvNotInitializedErr(hm.srv.State())
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

// handle calls a jrpc2.Func compatible function
func handle(ctx context.Context, req *jrpc2.Request, fn interface{}) (interface{}, error) {
	f := rpch.New(fn)
	return f.Handle(ctx, req)
}
