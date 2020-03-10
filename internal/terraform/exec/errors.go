package exec

import (
	"fmt"
	"os/exec"
	"reflect"
	"time"
)

type ExitError struct {
	Err    *exec.ExitError
	CtxErr error

	Path   string
	Stdout string
	Stderr string
}

func (e *ExitError) Unwrap() error {
	return e.CtxErr
}

func (e *ExitError) Error() string {
	out := fmt.Sprintf("terraform (pid %d) exited (code %d): %s\nstdout: %q\nstderr: %q",
		e.Err.Pid(),
		e.Err.ExitCode(),
		e.Err.ProcessState.String(),
		e.Stdout,
		e.Stderr)

	if e.CtxErr != nil {
		return fmt.Sprintf("%s.\n%s", e.CtxErr, e.Err)
	}

	return out
}

type execTimeoutErr struct {
	args     []string
	duration time.Duration
}

func (e *execTimeoutErr) Is(target error) bool {
	return reflect.DeepEqual(e, target)
}

func (e *execTimeoutErr) Error() string {
	return fmt.Sprintf("Execution of %q timed out after %s",
		e.args, e.duration)
}

func ExecTimeoutError(args []string, duration time.Duration) *execTimeoutErr {
	return &execTimeoutErr{args, duration}
}

type execCanceledErr struct {
	cmd []string
}

func (e *execCanceledErr) Is(target error) bool {
	return reflect.DeepEqual(e, target)
}

func (e *execCanceledErr) Error() string {
	return fmt.Sprintf("Execution of %q canceled", e.cmd)
}

func ExecCanceledError(cmd []string) *execCanceledErr {
	return &execCanceledErr{cmd}
}
