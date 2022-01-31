package job

import (
	"context"
	"fmt"
)

type JobStore interface {
	EnqueueJob(newJob Job) (ID, error)
	WaitForJobs(ctx context.Context, ids ...ID) error
}

type jobStoreCtxKey struct{}

func WithJobStore(ctx context.Context, js JobStore) context.Context {
	return context.WithValue(ctx, jobStoreCtxKey{}, js)
}

func JobStoreFromContext(ctx context.Context) (JobStore, error) {
	js, ok := ctx.Value(jobStoreCtxKey{}).(JobStore)
	if !ok {
		return nil, fmt.Errorf("not found JobStore in context")
	}
	return js, nil
}
