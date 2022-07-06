package job

import (
	"context"
)

type JobStore interface {
	EnqueueJob(newJob Job) (ID, error)
	WaitForJobs(ctx context.Context, ids ...ID) error
}
