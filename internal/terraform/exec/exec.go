package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/logging"
)

var defaultExecTimeout = 30 * time.Second

// We pass through all variables, but longer term we'll need to reflect
// that some variables might be workspace/directory specific
// and passing through these may be dangerous once the LS
// starts to execute commands which can mutate the state
var passthroughEnvVars = os.Environ()

// cmdCtxFunc allows mocking of Terraform in tests while retaining
// ability to pass context for timeout/cancellation
type cmdCtxFunc func(context.Context, string, ...string) *exec.Cmd

// ExecutorFactory can be used in external consumers of exec pkg
// to enable easy swapping with MockExecutor
type ExecutorFactory func(path string) *Executor

type Executor struct {
	timeout time.Duration

	execPath    string
	workDir     string
	logger      *log.Logger
	execLogPath string

	cmdCtxFunc cmdCtxFunc
}

type command struct {
	Cmd          *exec.Cmd
	Context      context.Context
	CancelFunc   context.CancelFunc
	StdoutBuffer *bytes.Buffer
	StderrBuffer *bytes.Buffer
}

func NewExecutor(path string) *Executor {
	return &Executor{
		timeout:  defaultExecTimeout,
		execPath: path,
		logger:   log.New(ioutil.Discard, "", 0),
		cmdCtxFunc: func(ctx context.Context, path string, arg ...string) *exec.Cmd {
			return exec.CommandContext(ctx, path, arg...)
		},
	}
}

func (e *Executor) SetLogger(logger *log.Logger) {
	e.logger = logger
}

func (e *Executor) SetExecLogPath(path string) {
	e.execLogPath = path
}

func (e *Executor) SetTimeout(duration time.Duration) {
	e.timeout = duration
}

func (e *Executor) SetWorkdir(workdir string) {
	e.workDir = workdir
}

func (e *Executor) GetExecPath() string {
	return e.execPath
}

func (e *Executor) cmd(ctx context.Context, args ...string) (*command, error) {
	if e.workDir == "" {
		return nil, fmt.Errorf("no work directory set")
	}

	cancel := func() {}
	if e.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer

	args = append(args[:1], append([]string{"-no-color"}, args[1:]...)...)
	cmd := e.cmdCtxFunc(ctx, e.execPath, args...)
	cmd.Args = append([]string{"terraform"}, args...)
	cmd.Dir = e.workDir
	cmd.Stderr = &errBuf
	cmd.Stdout = &outBuf

	// We don't perform upgrade from the context of executor
	// and don't report outdated version to users,
	// so we don't need to ask checkpoint for upgrades.
	cmd.Env = append(cmd.Env, "CHECKPOINT_DISABLE=1")

	for _, envVar := range passthroughEnvVars {
		cmd.Env = append(cmd.Env, envVar)
	}

	if e.execLogPath != "" {
		logPath, err := logging.ParseExecLogPath(cmd.Args, e.execLogPath)
		if err != nil {
			return &command{
				Cmd:          cmd,
				Context:      ctx,
				CancelFunc:   cancel,
				StdoutBuffer: &outBuf,
				StderrBuffer: &errBuf,
			}, fmt.Errorf("failed to parse log path: %w", err)
		}
		cmd.Env = append(cmd.Env, "TF_LOG=TRACE")
		cmd.Env = append(cmd.Env, "TF_LOG_PATH="+logPath)

		e.logger.Printf("Execution will be logged to %s", logPath)
	}
	return &command{
		Cmd:          cmd,
		Context:      ctx,
		CancelFunc:   cancel,
		StdoutBuffer: &outBuf,
		StderrBuffer: &errBuf,
	}, nil
}

func (e *Executor) waitCmd(command *command) ([]byte, error) {
	args := command.Cmd.Args
	e.logger.Printf("Waiting for command to finish ...")
	err := command.Cmd.Wait()
	if err != nil {
		if tErr, ok := err.(*exec.ExitError); ok {
			exitErr := &ExitError{
				Err:    tErr,
				Path:   command.Cmd.Path,
				Stdout: command.StdoutBuffer.String(),
				Stderr: command.StderrBuffer.String(),
			}

			ctxErr := command.Context.Err()
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				exitErr.CtxErr = ExecTimeoutError(args, e.timeout)
			}
			if errors.Is(ctxErr, context.Canceled) {
				exitErr.CtxErr = ExecCanceledError(args)
			}

			return nil, exitErr
		}

		return nil, err
	}

	pc := command.Cmd.ProcessState
	e.logger.Printf("terraform run (%s %q, in %q, pid %d) finished with exit code %d",
		e.execPath, args, e.workDir, pc.Pid(), pc.ExitCode())

	return command.StdoutBuffer.Bytes(), nil
}

func (e *Executor) runCmd(command *command) ([]byte, error) {
	args := command.Cmd.Args
	e.logger.Printf("Starting %s %q in %q...", e.execPath, args, e.workDir)
	err := command.Cmd.Start()
	if err != nil {
		return nil, err
	}

	return e.waitCmd(command)
}

func (e *Executor) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd, err := e.cmd(ctx, args...)
	e.logger.Printf("running with timeout %s", e.timeout)
	defer cmd.CancelFunc()
	if err != nil {
		return nil, err
	}
	return e.runCmd(cmd)
}

type Formatter func(ctx context.Context, input []byte) ([]byte, error)

func (e *Executor) FormatterForVersion(v string) (Formatter, error) {
	if v == "" {
		return nil, fmt.Errorf("unknown version - unable to provide formatter")
	}

	ver, err := version.NewVersion(v)
	if err != nil {
		return nil, err
	}

	// "fmt" command was first introduced in v0.7.7
	fmtCapableVersion, err := version.NewVersion("0.7.7")
	if err != nil {
		return nil, err
	}

	if ver.GreaterThanOrEqual(fmtCapableVersion) {
		return e.Format, nil
	}

	return nil, fmt.Errorf("no formatter available for %s", v)
}

func (e *Executor) Format(ctx context.Context, input []byte) ([]byte, error) {
	cmd, err := e.cmd(ctx, "fmt", "-")
	if err != nil {
		return nil, err
	}

	stdin, err := cmd.Cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Cmd.Start()
	if err != nil {
		return nil, err
	}

	_, err = writeAndClose(stdin, input)
	if err != nil {
		return nil, err
	}

	out, err := e.waitCmd(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to format: %w", err)
	}

	return out, nil
}

func writeAndClose(w io.WriteCloser, input []byte) (int, error) {
	defer w.Close()

	n, err := w.Write(input)
	if err != nil {
		return n, err
	}

	return n, nil
}

func (e *Executor) Version(ctx context.Context) (string, error) {
	out, err := e.run(ctx, "version")
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

func (e *Executor) VersionIsSupported(ctx context.Context, c version.Constraints) error {
	v, err := e.Version(ctx)
	if err != nil {
		return err
	}
	ver, err := version.NewVersion(v)
	if err != nil {
		return err
	}

	if !c.Check(ver) {
		return fmt.Errorf("version %s not supported (%s)",
			ver.String(), c.String())
	}

	return nil
}

func (e *Executor) ProviderSchemas(ctx context.Context) (*tfjson.ProviderSchemas, error) {
	outBytes, err := e.run(ctx, "providers", "schema", "-json")
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
