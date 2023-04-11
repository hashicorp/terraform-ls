// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

	Method string
}

func (e *ExitError) Unwrap() error {
	return e.CtxErr
}

func (e *ExitError) Error() string {
	out := fmt.Sprintf("terraform %q (pid %d) exited (code %d): %s",
		e.Method,
		e.Err.Pid(),
		e.Err.ExitCode(),
		e.Err.ProcessState.String())
	if e.CtxErr != nil {
		return fmt.Sprintf("%s.\n%s", e.CtxErr, e.Err)
	}

	return out
}

type execTimeoutErr struct {
	method   string
	duration time.Duration
}

func (e *execTimeoutErr) Is(target error) bool {
	return reflect.DeepEqual(e, target)
}

func (e *execTimeoutErr) Error() string {
	return fmt.Sprintf("Execution of %q timed out after %s",
		e.method, e.duration)
}

func ExecTimeoutError(method string, duration time.Duration) *execTimeoutErr {
	return &execTimeoutErr{method, duration}
}

type execCanceledErr struct {
	method string
}

func (e *execCanceledErr) Is(target error) bool {
	return reflect.DeepEqual(e, target)
}

func (e *execCanceledErr) Error() string {
	return fmt.Sprintf("Execution of %q canceled", e.method)
}

func ExecCanceledError(method string) *execCanceledErr {
	return &execCanceledErr{method}
}
