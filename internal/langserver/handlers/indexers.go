package handlers

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/state"
)

type closedDirJobStore struct {
	js *state.JobStore
}

func (js *closedDirJobStore) EnqueueJob(newJob job.Job) (job.ID, error) {
	return js.js.EnqueueJob(newJob)
}

func (js *closedDirJobStore) AwaitNextJob(ctx context.Context) (job.ID, job.Job, error) {
	return js.js.AwaitNextJob(ctx, false)
}

func (js *closedDirJobStore) FinishJob(id job.ID, jobErr error, deferredJobIds ...job.ID) error {
	return js.js.FinishJob(id, jobErr, deferredJobIds...)
}

func (js *closedDirJobStore) WaitForJobs(ctx context.Context, jobIds ...job.ID) error {
	return js.js.WaitForJobs(ctx, jobIds...)
}

type openDirJobStore struct {
	js *state.JobStore
}

func (js *openDirJobStore) EnqueueJob(newJob job.Job) (job.ID, error) {
	return js.js.EnqueueJob(newJob)
}

func (js *openDirJobStore) AwaitNextJob(ctx context.Context) (job.ID, job.Job, error) {
	return js.js.AwaitNextJob(ctx, true)
}

func (js *openDirJobStore) FinishJob(id job.ID, jobErr error, deferredJobIds ...job.ID) error {
	return js.js.FinishJob(id, jobErr, deferredJobIds...)
}

func (js *openDirJobStore) WaitForJobs(ctx context.Context, jobIds ...job.ID) error {
	return js.js.WaitForJobs(ctx, jobIds...)
}
