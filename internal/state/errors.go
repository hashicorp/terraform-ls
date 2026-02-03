// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/document"
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

type RecordNotFoundError struct {
	Source string
}

func (e *RecordNotFoundError) Error() string {
	msg := "record not found"
	if e.Source != "" {
		return fmt.Sprintf("%s: %s", e.Source, msg)
	}

	return msg
}

func IsRecordNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*RecordNotFoundError)
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

type walkerPathNotFound struct {
	Dir document.DirHandle
}

func (e walkerPathNotFound) Error() string {
	if e.Dir.URI != "" {
		return fmt.Sprintf("dir %q not found", e.Dir)
	}
	return "dir not found"
}

type pathAlreadyWalking struct {
	Dir document.DirHandle
}

func (e pathAlreadyWalking) Error() string {
	if e.Dir.URI != "" {
		return fmt.Sprintf("dir %q is already being walked", e.Dir)
	}
	return "dir is already being walked"
}
