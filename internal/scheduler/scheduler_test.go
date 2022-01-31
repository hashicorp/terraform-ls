package scheduler

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/state"
)

func TestScheduler_basic(t *testing.T) {
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss.SetLogger(testLogger())

	tmpDir := t.TempDir()

	ctx := context.Background()

	s := NewScheduler(&closedDirJobs{js: ss.JobStore}, 2)
	s.SetLogger(testLogger())
	s.Start(ctx)
	t.Cleanup(func() {
		s.Stop()
	})

	var jobsExecuted int64 = 0
	jobsToExecute := 50

	ids := make([]job.ID, 0)
	for i := 0; i < jobsToExecute; i++ {
		i := i
		dirPath := filepath.Join(tmpDir, fmt.Sprintf("folder-%d", i))

		newId, err := ss.JobStore.EnqueueJob(job.Job{
			Func: func(c context.Context) error {
				atomic.AddInt64(&jobsExecuted, 1)
				return nil
			},
			Dir:  document.DirHandleFromPath(dirPath),
			Type: "test-type",
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, newId)
	}

	err = ss.JobStore.WaitForJobs(ctx, ids...)
	if err != nil {
		t.Fatal(err)
	}

	if jobsExecuted != int64(jobsToExecute) {
		t.Fatalf("expected %d jobs to execute, got: %d", jobsToExecute, jobsExecuted)
	}
}

func TestScheduler_defer(t *testing.T) {
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss.SetLogger(testLogger())

	tmpDir := t.TempDir()

	ctx := context.Background()

	s := NewScheduler(&closedDirJobs{js: ss.JobStore}, 2)
	s.SetLogger(testLogger())
	s.Start(ctx)
	t.Cleanup(func() {
		s.Stop()
	})

	var jobsExecuted, deferredJobsExecuted int64 = 0, 0
	jobsToExecute := 50

	ids := make(job.IDs, 0)
	for i := 0; i < jobsToExecute; i++ {
		i := i
		dirPath := filepath.Join(tmpDir, fmt.Sprintf("folder-%d", i))

		newId, err := ss.JobStore.EnqueueJob(job.Job{
			Func: func(c context.Context) error {
				atomic.AddInt64(&jobsExecuted, 1)
				return nil
			},
			Dir:  document.DirHandleFromPath(dirPath),
			Type: "test-type",
			Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
				je, err := job.JobStoreFromContext(ctx)
				if err != nil {
					log.Fatal(err)
					return nil
				}

				id1, err := je.EnqueueJob(job.Job{
					Dir:  document.DirHandleFromPath(dirPath),
					Type: "test-1",
					Func: func(c context.Context) error {
						atomic.AddInt64(&deferredJobsExecuted, 1)
						return nil
					},
				})
				if err != nil {
					log.Fatal(err)
					return nil
				}
				ids = append(ids, id1)

				id2, err := je.EnqueueJob(job.Job{
					Dir:  document.DirHandleFromPath(dirPath),
					Type: "test-2",
					Func: func(c context.Context) error {
						atomic.AddInt64(&deferredJobsExecuted, 1)
						return nil
					},
				})
				if err != nil {
					log.Fatal(err)
					return nil
				}
				ids = append(ids, id2)

				return
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, newId)
	}

	err = ss.JobStore.WaitForJobs(ctx, ids...)
	if err != nil {
		t.Fatal(err)
	}

	if jobsExecuted != int64(jobsToExecute) {
		t.Fatalf("expected %d jobs to execute, got: %d", jobsToExecute, jobsExecuted)
	}

	expectedDeferredJobs := int64(jobsToExecute * 2)
	if deferredJobsExecuted != expectedDeferredJobs {
		t.Fatalf("expected %d deferred jobs to execute, got: %d", expectedDeferredJobs, deferredJobsExecuted)
	}
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}

type closedDirJobs struct {
	js *state.JobStore
}

func (js *closedDirJobs) EnqueueJob(newJob job.Job) (job.ID, error) {
	return js.js.EnqueueJob(newJob)
}

func (js *closedDirJobs) AwaitNextJob(ctx context.Context) (job.ID, job.Job, error) {
	return js.js.AwaitNextJob(ctx, false)
}

func (js *closedDirJobs) FinishJob(id job.ID, jobErr error, deferredJobIds ...job.ID) error {
	return js.js.FinishJob(id, jobErr, deferredJobIds...)
}

func (js *closedDirJobs) WaitForJobs(ctx context.Context, jobIds ...job.ID) error {
	return js.js.WaitForJobs(ctx, jobIds...)
}
