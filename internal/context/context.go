package context

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	"github.com/sourcegraph/go-lsp"
)

type contextKey struct {
	Name string
}

func (k *contextKey) String() string {
	return k.Name
}

var (
	ctxFs               = &contextKey{"filesystem"}
	ctxTerraformExec    = &contextKey{"terraform executor"}
	ctxClientCapsSetter = &contextKey{"client capabilities setter"}
	ctxClientCaps       = &contextKey{"client capabilities"}
	ctxTfSchemaWriter   = &contextKey{"schema writer"}
	ctxTfSchemaReader   = &contextKey{"schema reader"}
	ctxTfVersion        = &contextKey{"terraform version"}
	ctxTfVersionSetter  = &contextKey{"terraform version setter"}
	ctxTfExecLogPath    = &contextKey{"terraform executor log path"}
	ctxTfExecTimeout    = &contextKey{"terraform execution timeout"}
)

func missingContextErr(ctxKey *contextKey) *MissingContextErr {
	return &MissingContextErr{ctxKey}
}

func WithFilesystem(fs filesystem.Filesystem, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxFs, fs)
}

func Filesystem(ctx context.Context) (filesystem.Filesystem, error) {
	fs, ok := ctx.Value(ctxFs).(filesystem.Filesystem)
	if !ok {
		return nil, missingContextErr(ctxFs)
	}

	return fs, nil
}

func WithTerraformExecutor(tf *exec.Executor, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTerraformExec, tf)
}

func TerraformExecutor(ctx context.Context) (*exec.Executor, error) {
	tf, ok := ctx.Value(ctxTerraformExec).(*exec.Executor)
	if !ok {
		return nil, missingContextErr(ctxTerraformExec)
	}

	return tf, nil
}

func WithClientCapabilitiesSetter(caps *lsp.ClientCapabilities, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxClientCapsSetter, caps)
}

func SetClientCapabilities(ctx context.Context, caps *lsp.ClientCapabilities) error {
	cc, ok := ctx.Value(ctxClientCapsSetter).(*lsp.ClientCapabilities)
	if !ok {
		return missingContextErr(ctxClientCapsSetter)
	}

	*cc = *caps
	return nil
}

func WithClientCapabilities(caps *lsp.ClientCapabilities, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxClientCaps, caps)
}

func ClientCapabilities(ctx context.Context) (lsp.ClientCapabilities, error) {
	caps, ok := ctx.Value(ctxClientCaps).(*lsp.ClientCapabilities)
	if !ok {
		return lsp.ClientCapabilities{}, missingContextErr(ctxClientCaps)
	}

	return *caps, nil
}

func WithTerraformSchemaWriter(s schema.Writer, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfSchemaWriter, s)
}

func TerraformSchemaWriter(ctx context.Context) (schema.Writer, error) {
	ss, ok := ctx.Value(ctxTfSchemaWriter).(schema.Writer)
	if !ok {
		return nil, missingContextErr(ctxTfSchemaWriter)
	}

	return ss, nil
}

func WithTerraformSchemaReader(s schema.Reader, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfSchemaReader, s)
}

func TerraformSchemaReader(ctx context.Context) (schema.Reader, error) {
	ss, ok := ctx.Value(ctxTfSchemaReader).(schema.Reader)
	if !ok {
		return nil, missingContextErr(ctxTfSchemaReader)
	}

	return ss, nil
}

func WithTerraformVersion(v string, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfVersion, v)
}

func TerraformVersion(ctx context.Context) (string, error) {
	tfv, ok := ctx.Value(ctxTfVersion).(string)
	if !ok {
		return "", missingContextErr(ctxTfVersion)
	}

	return tfv, nil
}

func WithTerraformVersionSetter(v *string, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfVersionSetter, v)
}

func SetTerraformVersion(ctx context.Context, v string) error {
	tfv, ok := ctx.Value(ctxTfVersionSetter).(*string)
	if !ok {
		return missingContextErr(ctxTfVersionSetter)
	}
	*tfv = v

	return nil
}

func WithTerraformExecLogPath(path string, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfExecLogPath, path)
}

func TerraformExecLogPath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(ctxTfExecLogPath).(string)
	return path, ok
}

func WithTerraformExecTimeout(timeout time.Duration, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfExecTimeout, timeout)
}

func TerraformExecTimeout(ctx context.Context) (time.Duration, bool) {
	path, ok := ctx.Value(ctxTfExecTimeout).(time.Duration)
	return path, ok
}
