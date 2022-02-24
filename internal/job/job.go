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

	// Defer is a function to execute after Func is executed
	// and before the job is marked as done (StateDone).
	// This can be used to schedule jobs dependent on the main job.
	Defer DeferFunc
}

// DeferFunc represents a deferred function scheduling more jobs
// based on jobErr (any error returned from the main job).
// Newly queued job IDs should be returned to allow for synchronization.
type DeferFunc func(ctx context.Context, jobErr error) IDs

func (job Job) Copy() Job {
	return Job{
		Func:  job.Func,
		Dir:   job.Dir,
		Type:  job.Type,
		Defer: job.Defer,
	}
}
