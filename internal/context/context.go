package context

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	"github.com/sourcegraph/go-lsp"
)

func WithSignalCancel(ctx context.Context, l *log.Logger, sigs ...os.Signal) (
	context.Context, context.CancelFunc) {
	ctx, cancelFunc := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, sigs...)

	go func() {
		select {
		case sig := <-sigChan:
			l.Printf("Cancellation signal (%s) received", sig)
			cancelFunc()
		case <-ctx.Done():
		}
	}()

	f := func() {
		signal.Stop(sigChan)
		cancelFunc()
	}

	return ctx, f
}

const ctxFs = "ctxFilesystem"

func WithFilesystem(fs filesystem.Filesystem, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxFs, fs)
}

func Filesystem(ctx context.Context) (filesystem.Filesystem, error) {
	fs, ok := ctx.Value(ctxFs).(filesystem.Filesystem)
	if !ok {
		return nil, fmt.Errorf("no filesystem")
	}

	return fs, nil
}

const ctxTerraformExec = "ctxTerraformExec"

func WithTerraformExecutor(tf *exec.Executor, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTerraformExec, tf)
}

func TerraformExecutor(ctx context.Context) (*exec.Executor, error) {
	tf, ok := ctx.Value(ctxTerraformExec).(*exec.Executor)
	if !ok {
		return nil, fmt.Errorf("no terraform executor")
	}

	return tf, nil
}

const ctxClientCapsSetter = "ctxClientCapabilitiesSetter"

func WithClientCapabilitiesSetter(caps *lsp.ClientCapabilities, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxClientCapsSetter, caps)
}

func SetClientCapabilities(ctx context.Context, caps *lsp.ClientCapabilities) error {
	cc, ok := ctx.Value(ctxClientCapsSetter).(*lsp.ClientCapabilities)
	if !ok {
		return fmt.Errorf("no client capabilities setter")
	}

	*cc = *caps
	return nil
}

const ctxClientCaps = "ctxClientCapabilities"

func WithClientCapabilities(caps *lsp.ClientCapabilities, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxClientCaps, caps)
}

func ClientCapabilities(ctx context.Context) (lsp.ClientCapabilities, error) {
	caps, ok := ctx.Value(ctxClientCaps).(*lsp.ClientCapabilities)
	if !ok {
		return lsp.ClientCapabilities{}, fmt.Errorf("no client capabilities")
	}

	return *caps, nil
}

const ctxTfSchemaWriter = "ctxTerraformSchemaWriter"

func WithTerraformSchemaWriter(s schema.Writer, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfSchemaWriter, s)
}

func TerraformSchemaWriter(ctx context.Context) (schema.Writer, error) {
	ss, ok := ctx.Value(ctxTfSchemaWriter).(schema.Writer)
	if !ok {
		return nil, fmt.Errorf("no terraform schema writer")
	}

	return ss, nil
}

const ctxTfSchemaReader = "ctxTerraformSchemaWriter"

func WithTerraformSchemaReader(s schema.Reader, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfSchemaReader, s)
}

func TerraformSchemaReader(ctx context.Context) (schema.Reader, error) {
	ss, ok := ctx.Value(ctxTfSchemaReader).(schema.Reader)
	if !ok {
		return nil, fmt.Errorf("no terraform schema reader")
	}

	return ss, nil
}

const ctxTfVersion = "ctxTerraformVersion"

func WithTerraformVersion(v string, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfVersion, v)
}

func TerraformVersion(ctx context.Context) (string, error) {
	tfv, ok := ctx.Value(ctxTfVersion).(string)
	if !ok {
		return "", fmt.Errorf("no Terraform version")
	}

	return tfv, nil
}

const ctxTfVersionSetter = "ctxTerraformVersionSetter"

func WithTerraformVersionSetter(v *string, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfVersionSetter, v)
}

func SetTerraformVersion(ctx context.Context, v string) error {
	tfv, ok := ctx.Value(ctxTfVersionSetter).(*string)
	if !ok {
		return fmt.Errorf("no Terraform version setter")
	}
	*tfv = v

	return nil
}

const ctxTfExecLogPath = "ctxTerraformExecLogPath"

func WithTerraformExecLogPath(path string, ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxTfExecLogPath, path)
}

func TerraformExecLogPath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(ctxTfExecLogPath).(string)
	return path, ok
}
