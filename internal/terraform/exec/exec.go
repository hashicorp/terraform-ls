package exec

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os/exec"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/logging"
)

var defaultExecTimeout = 30 * time.Second

type Executor struct {
	tf         *tfexec.Terraform
	timeout    time.Duration
	rawLogPath string
}

func NewExecutor(workDir, execPath string) (TerraformExecutor, error) {
	tf, err := tfexec.NewTerraform(workDir, execPath)
	if err != nil {
		return nil, err
	}
	return &Executor{
		timeout: defaultExecTimeout,
		tf:      tf,
	}, nil
}

func (e *Executor) SetLogger(logger *log.Logger) {
	e.tf.SetLogger(logger)
}

func (e *Executor) SetExecLogPath(rawPath string) error {
	e.rawLogPath = rawPath
	return nil
}

func (e *Executor) SetTimeout(duration time.Duration) {
	e.timeout = duration
}

func (e *Executor) GetExecPath() string {
	return e.tf.ExecPath()
}

func (e *Executor) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, e.timeout)
}

func (e *Executor) contextfulError(ctx context.Context, method string, err error) error {
	exitErr, ok := err.(*exec.ExitError)
	if ok {
		return &ExitError{
			Err:    exitErr,
			CtxErr: e.enrichCtxErr(method, ctx.Err()),
			Method: method,
		}
	}
	return e.enrichCtxErr(method, err)
}

func (e *Executor) enrichCtxErr(method string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ExecTimeoutError(method, e.timeout)
	}
	if errors.Is(err, context.Canceled) {
		return ExecCanceledError(method)
	}
	return err
}

func (e *Executor) setLogPath(method string) error {
	logPath, err := logging.ParseExecLogPath(method, e.rawLogPath)
	if err != nil {
		return err
	}
	return e.tf.SetLogPath(logPath)
}

func (e *Executor) Init(ctx context.Context, opts ...tfexec.InitOption) error {
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()
	err := e.setLogPath("Init")
	if err != nil {
		return err
	}

	return e.contextfulError(ctx, "Init", e.tf.Init(ctx, opts...))
}

func (e *Executor) Format(ctx context.Context, input []byte) ([]byte, error) {
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()
	err := e.setLogPath("Format")
	if err != nil {
		return nil, err
	}

	br := bytes.NewReader(input)
	buf := bytes.NewBuffer([]byte{})

	err = e.tf.Format(ctx, br, buf)

	return buf.Bytes(), e.contextfulError(ctx, "Format", err)
}

func (e *Executor) Version(ctx context.Context) (*version.Version, map[string]*version.Version, error) {
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()
	err := e.setLogPath("Version")
	if err != nil {
		return nil, nil, err
	}

	ver, pv, err := e.tf.Version(ctx, true)
	return ver, pv, e.contextfulError(ctx, "Version", err)
}

func (e *Executor) ProviderSchemas(ctx context.Context) (*tfjson.ProviderSchemas, error) {
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()
	err := e.setLogPath("ProviderSchemas")
	if err != nil {
		return nil, err
	}

	ps, err := e.tf.ProvidersSchema(ctx)
	return ps, e.contextfulError(ctx, "ProviderSchemas", err)
}
