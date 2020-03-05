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

type Executor struct {
	ctx     context.Context
	timeout time.Duration
	workDir string
	logger  *log.Logger
}

func NewExecutor(ctx context.Context) *Executor {
	return &Executor{
		ctx:     ctx,
		timeout: 10 * time.Second,
		logger:  log.New(ioutil.Discard, "", 0),
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

	allArgs := []string{"terraform"}
	allArgs = append(allArgs, args...)

	var outBuf bytes.Buffer
	var errBuf strings.Builder

	path, err := exec.LookPath("terraform")
	if err != nil {
		e.logger.Printf("[ERROR] Unable to find terraform with PATH set to %q", os.Getenv("PATH"))
		return nil, fmt.Errorf("unable to find terraform for %q: %s", e.workDir, err)
	}

	cmd := exec.CommandContext(ctx, path)

	cmd.Args = allArgs
	cmd.Dir = e.workDir
	cmd.Stderr = &errBuf
	cmd.Stdout = &outBuf

	e.logger.Printf("Running %s %q in %q...", path, allArgs[1:], e.workDir)
	err = cmd.Run()
	if err != nil {
		if tErr, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("terraform (pid %d) exited (code %d): %s\nstdout: %q\nstderr: %q",
				tErr.Pid(),
				tErr.ExitCode(),
				tErr.ProcessState.String(),
				outBuf.String(),
				errBuf.String())
		}

		ctxErr := ctx.Err()
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w.\n%s",
				ExecTimeoutError(allArgs, e.timeout), err)
		}
		if errors.Is(ctxErr, context.Canceled) {
			return nil, fmt.Errorf("%w.\n%s",
				ExecCanceledError(allArgs), err)
		}

		return nil, err
	}

	pc := cmd.ProcessState
	e.logger.Printf("terraform run (%s %q, in %q, pid %d) finished with exit code %d",
		path, allArgs[1:], e.workDir, pc.Pid(), pc.ExitCode())

	return outBuf.Bytes(), nil
}

func (e *Executor) Version() (string, error) {
	out, err := e.run("version")
	if err != nil {
		return "", fmt.Errorf("failed to get version: %s", err)
	}

	return string(out), nil
}

func (e *Executor) ProviderSchemas() (*tfjson.ProviderSchemas, error) {
	outBytes, err := e.run("providers", "schema", "-json")
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %s", err)
	}

	var schemas tfjson.ProviderSchemas
	err = json.Unmarshal(outBytes, &schemas)
	if err != nil {
		return nil, err
	}

	return &schemas, nil
}

type execTimeoutErr struct {
	cmd      []string
	duration time.Duration
}

func (e *execTimeoutErr) Error() string {
	return fmt.Sprintf("Execution of %q timed out after %s",
		e.cmd, e.duration)
}

func ExecTimeoutError(cmd []string, duration time.Duration) *execTimeoutErr {
	return &execTimeoutErr{cmd, duration}
}

type execCanceledErr struct {
	cmd []string
}

func (e *execCanceledErr) Error() string {
	return fmt.Sprintf("Execution of %q canceled", e.cmd)
}

func ExecCanceledError(cmd []string) *execCanceledErr {
	return &execCanceledErr{cmd}
}
