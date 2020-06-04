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
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	"github.com/hashicorp/terraform-ls/langserver/session"
	"github.com/sourcegraph/go-lsp"
)

type service struct {
	logger *log.Logger

	srvCtx context.Context

	sessCtx     context.Context
	stopSession context.CancelFunc

	tfDiscoFunc  discovery.DiscoveryFunc
	ss           *schema.Storage
	executorFunc func(ctx context.Context, execPath string) *exec.Executor
}

var discardLogs = log.New(ioutil.Discard, "", 0)

func NewSession(srvCtx context.Context) session.Session {
	sessCtx, stopSession := context.WithCancel(srvCtx)
	d := &discovery.Discovery{}
	return &service{
		logger:       discardLogs,
		srvCtx:       srvCtx,
		sessCtx:      sessCtx,
		stopSession:  stopSession,
		executorFunc: exec.NewExecutor,
		tfDiscoFunc:  d.LookPath,
		ss:           schema.NewStorage(),
	}
}

func (svc *service) SetLogger(logger *log.Logger) {
	svc.logger = logger
}

func (svc *service) SetDiscoveryFunc(f discovery.DiscoveryFunc) {
	svc.tfDiscoFunc = f
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
	tfVersion := "0.0.0"
	svc.ss.SetLogger(svc.logger)

	m := map[string]rpch.Func{
		"initialize": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.Initialize(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			ctx = lsctx.WithClientCapabilitiesSetter(cc, ctx)

			tfPath, err := svc.tfDiscoFunc()
			if err != nil {
				return nil, err
			}

			// We intentionally pass session context here to make executor cancellable
			// on session shutdown, rather than response delivery or request cancellation
			// as some operations may run in isolated goroutines
			tf := svc.executorFunc(svc.sessCtx, tfPath)

			// Log path is set via CLI flag, hence in the server context
			if path, ok := lsctx.TerraformExecLogPath(svc.srvCtx); ok {
				tf.SetExecLogPath(path)
			}

			// Timeout is set via CLI flag, hence in the server context
			if timeout, ok := lsctx.TerraformExecTimeout(svc.srvCtx); ok {
				tf.SetTimeout(timeout)
			}

			tf.SetLogger(svc.logger)

			ctx = lsctx.WithTerraformExecutor(tf, ctx)
			ctx = lsctx.WithTerraformVersionSetter(&tfVersion, ctx)
			ctx = lsctx.WithTerraformSchemaWriter(svc.ss, ctx)

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
			ctx = lsctx.WithTerraformVersion(tfVersion, ctx)
			ctx = lsctx.WithTerraformSchemaReader(svc.ss, ctx)

			return handle(ctx, req, lh.TextDocumentComplete)
		},
		"textDocument/formatting": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithFilesystem(fs, ctx)

			tfPath, err := svc.tfDiscoFunc()
			if err != nil {
				return nil, err
			}

			tf := svc.executorFunc(ctx, tfPath)
			// Log path is set via CLI flag, hence the server context
			if path, ok := lsctx.TerraformExecLogPath(svc.srvCtx); ok {
				tf.SetExecLogPath(path)
			}
			tf.SetLogger(svc.logger)

			ctx = lsctx.WithTerraformExecutor(tf, ctx)

			return handle(ctx, req, lh.TextDocumentFormatting)
		},
		"textDocument/hover": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := session.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithFilesystem(fs, ctx) // TODO: Read-only FS
			ctx = lsctx.WithClientCapabilities(cc, ctx)
			ctx = lsctx.WithTerraformVersion(tfVersion, ctx)
			ctx = lsctx.WithTerraformSchemaReader(svc.ss, ctx)

			return handle(ctx, req, lh.TextDocumentHover)
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

	svc.logger.Println("Stopping schema watcher for session ...")
	err := svc.ss.StopWatching()
	if err != nil {
		svc.logger.Println("Unable to stop schema watcher for session:", err)
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

func supportsTerraform(tfVersion string) error {
	err := schema.SchemaSupportsTerraform(tfVersion)
	if err != nil {
		return err
	}

	err = lang.ParserSupportsTerraform(tfVersion)
	if err != nil {
		return err
	}

	return nil
}
