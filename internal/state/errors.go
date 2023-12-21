// Copyright (c) HashiCorp, Inc.
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

type ModuleNotFoundError struct {
	Source string
}

func (e *ModuleNotFoundError) Error() string {
	msg := "module not found"
	if e.Source != "" {
		return fmt.Sprintf("%s: %s", e.Source, msg)
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

type VarsNotFoundError struct {
	Source string
}

func (e *VarsNotFoundError) Error() string {
	msg := "vars not found"
	if e.Source != "" {
		return fmt.Sprintf("%s: %s", e.Source, msg)
	}

	return msg
}

func IsVarsNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*VarsNotFoundError)
	return ok
}

func IsDirNotFound(err error) bool {
	if err == nil {
		return false
	}
	return IsModuleNotFound(err) || IsVarsNotFound(err)
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
