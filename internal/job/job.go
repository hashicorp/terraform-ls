// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
)

type Job struct {
	// Func represents the job to execute
	Func func(ctx context.Context) error

	// Dir describes the directory which the job belongs to,
	// which is used for deduplication of queued jobs (along with Type)
	// and prioritization
	Dir document.DirHandle

	// Type describes type of the job (e.g. GetTerraformVersion),
	// which is used for deduplication of queued jobs along with Dir.
	Type string

	// Priority represents priority with which the job should be scheduled.
	// This overrides the priority implied from whether the dir is open.
	Priority JobPriority

	// Defer is a function to execute after Func is executed
	// and before the job is marked as done (StateDone).
	// This can be used to schedule jobs dependent on the main job.
	Defer DeferFunc

	// DependsOn represents any other job IDs this job depends on.
	// This will be taken into account when scheduling, so that only
	// jobs with no dependencies are dispatched at any time.
	DependsOn IDs

	// IgnoreState indicates to the job (as defined by Func)
	// whether to ignore existing state, i.e. whether to invalidate cache.
	// It is up to [Func] to read this flag from ctx and reflect it.
	IgnoreState bool
}

// DeferFunc represents a deferred function scheduling more jobs
// based on jobErr (any error returned from the main job).
// Newly queued job IDs should be returned to allow for synchronization.
type DeferFunc func(ctx context.Context, jobErr error) (IDs, error)

func (job Job) Copy() Job {
	return Job{
		Func:        job.Func,
		Dir:         job.Dir,
		Type:        job.Type,
		Priority:    job.Priority,
		Defer:       job.Defer,
		IgnoreState: job.IgnoreState,
		DependsOn:   job.DependsOn.Copy(),
	}
}

type JobPriority int

const (
	LowPriority  JobPriority = -1
	HighPriority JobPriority = 1
)
