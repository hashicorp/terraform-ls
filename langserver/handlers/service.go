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
	"github.com/hashicorp/terraform-ls/langserver/svcctl"
	"github.com/sourcegraph/go-lsp"
)

type service struct {
	logger *log.Logger

	srvCtx context.Context

	svcCtx      context.Context
	svcStopFunc context.CancelFunc

	ss           *schema.Storage
	executorFunc func(ctx context.Context, execPath string) *exec.Executor
}

var discardLogs = log.New(ioutil.Discard, "", 0)

func NewService(srvCtx context.Context) svcctl.Service {
	svcCtx, stopSvc := context.WithCancel(srvCtx)
	return &service{
		logger:       discardLogs,
		srvCtx:       srvCtx,
		svcCtx:       svcCtx,
		svcStopFunc:  stopSvc,
		executorFunc: exec.NewExecutor,
		ss:           schema.NewStorage(),
	}
}

func (svc *service) SetLogger(logger *log.Logger) {
	svc.logger = logger
}

// Assigner builds out the jrpc2.Map according to the LSP protocol
// and passes related dependencies to handlers via context
func (svc *service) Assigner() (jrpc2.Assigner, error) {
	svc.logger.Println("Preparing new service ...")

	svcCtl := svcctl.NewServiceController(svc.svcStopFunc)

	err := svcCtl.Prepare()
	if err != nil {
		return nil, fmt.Errorf("Unable to prepare service: %w", err)
	}

	fs := filesystem.NewFilesystem()
	fs.SetLogger(svc.logger)
	lh := LogHandler(svc.logger)
	cc := &lsp.ClientCapabilities{}
	tfVersion := "0.0.0"
	svc.ss.SetLogger(svc.logger)

	m := map[string]rpch.Func{
		"initialize": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := svcCtl.Initialize(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			ctx = lsctx.WithClientCapabilitiesSetter(cc, ctx)

			tfPath, err := discovery.LookPath()
			if err != nil {
				return nil, err
			}

			// We intentionally pass service context here to make executor cancellable
			// on service shutdown, rather than response delivery or request cancellation
			// as some operations may run in isolated goroutines
			tf := svc.executorFunc(svc.svcCtx, tfPath)

			// Log path is set via CLI flag, hence in the server context
			if path, ok := lsctx.TerraformExecLogPath(svc.srvCtx); ok {
				tf.SetExecLogPath(path)
			}
			tf.SetLogger(svc.logger)

			ctx = lsctx.WithTerraformExecutor(tf, ctx)
			ctx = lsctx.WithTerraformVersionSetter(&tfVersion, ctx)
			ctx = lsctx.WithTerraformSchemaWriter(svc.ss, ctx)

			return handle(ctx, req, lh.Initialize)
		},
		"initialized": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := svcCtl.ConfirmInitialization(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)

			return handle(ctx, req, Initialized)
		},
		"textDocument/didChange": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := svcCtl.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidChange)
		},
		"textDocument/didOpen": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := svcCtl.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidOpen)
		},
		"textDocument/didClose": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := svcCtl.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, TextDocumentDidClose)
		},
		"textDocument/completion": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := svcCtl.CheckInitializationIsConfirmed()
			if err != nil {
				return nil, err
			}

			ctx = lsctx.WithFilesystem(fs, ctx) // TODO: Read-only FS
			ctx = lsctx.WithClientCapabilities(cc, ctx)
			ctx = lsctx.WithTerraformVersion(tfVersion, ctx)
			ctx = lsctx.WithTerraformSchemaReader(svc.ss, ctx)

			return handle(ctx, req, lh.TextDocumentComplete)
		},
		"shutdown": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := svcCtl.Shutdown(req)
			if err != nil {
				return nil, err
			}
			ctx = lsctx.WithFilesystem(fs, ctx)
			return handle(ctx, req, Shutdown)
		},
		"exit": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := svcCtl.Exit()
			if err != nil {
				return nil, err
			}

			svc.svcStopFunc()

			return nil, nil
		},
		"$/cancelRequest": func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
			err := svcCtl.CheckInitializationIsConfirmed()
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
		svc.logger.Printf("Service stopped unexpectedly (err: %v)", status.Err)
	}

	svc.logger.Println("Stopping schema watcher for service ...")
	err := svc.ss.StopWatching()
	if err != nil {
		svc.logger.Println("Unable to stop schema watcher for service:", err)
	}

	svc.svcStopFunc()
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
