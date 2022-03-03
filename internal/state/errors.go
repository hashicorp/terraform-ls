package state

import (
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/job"
)

type AlreadyExistsError struct {
	Idx string
}

func (e *AlreadyExistsError) Error() string {
	if e.Idx != "" {
		return fmt.Sprintf("%s already exists", e.Idx)
	}
	return "already exists"
}

type NoSchemaError struct{}

func (e *NoSchemaError) Error() string {
	return "no schema found"
}

type ModuleNotFoundError struct {
	Path string
}

func (e *ModuleNotFoundError) Error() string {
	msg := "module not found"
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, msg)
	}

	return msg
}

func IsModuleNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ModuleNotFoundError)
	return ok
}

type jobAlreadyRunning struct {
	ID job.ID
}

func (e jobAlreadyRunning) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("job %q is already running", e.ID)
	}
	return "job is already running"
}

type jobNotFound struct {
	ID job.ID
}

func (e jobNotFound) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("job %q not found", e.ID)
	}
	return "job not found"
}
