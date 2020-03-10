package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
)

type cmdCtxFunc func(context.Context, string, ...string) *exec.Cmd

type Executor struct {
	ctx     context.Context
	timeout time.Duration
	workDir string
	logger  *log.Logger

	cmdCtxFunc cmdCtxFunc
}

func NewExecutor(ctx context.Context) *Executor {
	return &Executor{
		ctx:     ctx,
		timeout: 10 * time.Second,
		logger:  log.New(ioutil.Discard, "", 0),
		cmdCtxFunc: func(ctx context.Context, path string, arg ...string) *exec.Cmd {
			return exec.CommandContext(ctx, path, arg...)
		},
	}
}

func (e *Executor) SetLogger(logger *log.Logger) {
	e.logger = logger
}

func (e *Executor) SetTimeout(duration time.Duration) {
	e.timeout = duration
}

func (e *Executor) SetWorkdir(workdir string) {
	e.workDir = workdir
}

func (e *Executor) run(args ...string) ([]byte, error) {
	ctx := e.ctx
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(e.ctx, e.timeout)
		defer cancel()
	}

	var outBuf bytes.Buffer
	var errBuf strings.Builder

	path, err := exec.LookPath("terraform")
	if err != nil {
		e.logger.Printf("[ERROR] Unable to find terraform with PATH set to %q", os.Getenv("PATH"))
		return nil, fmt.Errorf("unable to find terraform for %q: %s", e.workDir, err)
	}

	cmd := e.cmdCtxFunc(ctx, path, args...)
	cmd.Args = append([]string{"terraform"}, args...)
	cmd.Dir = e.workDir
	cmd.Stderr = &errBuf
	cmd.Stdout = &outBuf

	e.logger.Printf("Running %s %q in %q...", path, args, e.workDir)
	err = cmd.Run()
	if err != nil {
		if tErr, ok := err.(*exec.ExitError); ok {
			exitErr := &ExitError{
				Err:    tErr,
				Path:   cmd.Path,
				Stdout: outBuf.String(),
				Stderr: errBuf.String(),
			}

			ctxErr := ctx.Err()
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				exitErr.CtxErr = ExecTimeoutError(cmd.Args, e.timeout)
			}
			if errors.Is(ctxErr, context.Canceled) {
				exitErr.CtxErr = ExecCanceledError(args)
			}

			return nil, exitErr
		}

		return nil, err
	}

	pc := cmd.ProcessState
	e.logger.Printf("terraform run (%s %q, in %q, pid %d) finished with exit code %d",
		path, args, e.workDir, pc.Pid(), pc.ExitCode())

	return outBuf.Bytes(), nil
}

func (e *Executor) Version() (string, error) {
	out, err := e.run("version")
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}
	outString := string(out)
	lines := strings.Split(outString, "\n")
	if len(lines) < 1 {
		return "", fmt.Errorf("unexpected version output: %q", outString)
	}
	version := strings.TrimLeft(lines[0], "Terraform v")

	return version, nil
}

func (e *Executor) ProviderSchemas() (*tfjson.ProviderSchemas, error) {
	outBytes, err := e.run("providers", "schema", "-json")
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %w", err)
	}

	var schemas tfjson.ProviderSchemas
	err = json.Unmarshal(outBytes, &schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &schemas, nil
}
