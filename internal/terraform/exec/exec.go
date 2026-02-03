// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var defaultExecTimeout = 30 * time.Second

const tracerName = "github.com/hashicorp/terraform-ls/internal/terraform/exec"

type ctxKey string

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

func (e *Executor) setSpanStatus(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "execution returned error")
		return
	}
	span.SetStatus(codes.Ok, "execution successful")
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
	ctx, span := otel.Tracer(tracerName).Start(ctx, "terraform-exec:Init")
	defer span.End()

	err = e.tf.Init(ctx, opts...)
	e.setSpanStatus(span, err)

	return e.contextfulError(ctx, "Init", err)
}

func (e *Executor) Get(ctx context.Context, opts ...tfexec.GetCmdOption) error {
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()
	err := e.setLogPath("Get")
	if err != nil {
		return err
	}
	ctx, span := otel.Tracer(tracerName).Start(ctx, "terraform-exec:Get")
	defer span.End()

	err = e.tf.Get(ctx, opts...)
	e.setSpanStatus(span, err)

	return e.contextfulError(ctx, "Get", err)
}

func (e *Executor) Format(ctx context.Context, input []byte) ([]byte, error) {
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()
	err := e.setLogPath("Format")
	if err != nil {
		return nil, err
	}

	ctx, span := otel.Tracer(tracerName).Start(ctx, "terraform-exec:Format",
		trace.WithAttributes(attribute.KeyValue{
			Key:   attribute.Key("stdinLength"),
			Value: attribute.IntValue(len(input)),
		}))
	defer span.End()

	br := bytes.NewReader(input)
	buf := bytes.NewBuffer([]byte{})

	err = e.tf.Format(ctx, br, buf)
	e.setSpanStatus(span, err)

	return buf.Bytes(), e.contextfulError(ctx, "Format", err)
}

func (e *Executor) Validate(ctx context.Context) ([]tfjson.Diagnostic, error) {
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()
	err := e.setLogPath("Validate")
	if err != nil {
		return []tfjson.Diagnostic{}, err
	}

	ctx, span := otel.Tracer(tracerName).Start(ctx, "terraform-exec:Validate")
	defer span.End()

	validation, err := e.tf.Validate(ctx)
	e.setSpanStatus(span, err)
	if err != nil {
		return []tfjson.Diagnostic{}, e.contextfulError(ctx, "Validate", err)
	}

	return validation.Diagnostics, nil
}

func (e *Executor) Version(ctx context.Context) (*version.Version, map[string]*version.Version, error) {
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()
	err := e.setLogPath("Version")
	if err != nil {
		return nil, nil, err
	}

	ctx, span := otel.Tracer(tracerName).Start(ctx, "terraform-exec:Version")
	defer span.End()

	ver, pv, err := e.tf.Version(ctx, true)
	e.setSpanStatus(span, err)

	return ver, pv, e.contextfulError(ctx, "Version", err)
}

func (e *Executor) ProviderSchemas(ctx context.Context) (*tfjson.ProviderSchemas, error) {
	ctx, cancel := e.withTimeout(ctx)
	defer cancel()
	err := e.setLogPath("ProviderSchemas")
	if err != nil {
		return nil, err
	}

	ctx, span := otel.Tracer(tracerName).Start(ctx, "terraform-exec:ProviderSchemas")
	defer span.End()

	ps, err := e.tf.ProvidersSchema(ctx)
	e.setSpanStatus(span, err)

	return ps, e.contextfulError(ctx, "ProviderSchemas", err)
}
