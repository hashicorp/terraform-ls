package state

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
)

func TestJobStore_EnqueueJob(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	id1, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  document.DirHandleFromPath("/test-1"),
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  document.DirHandleFromPath("/test-2"),
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedIds := job.IDs{id1, id2}

	ids, err := ss.JobStore.ListQueuedJobs()
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(expectedIds, func(i, j int) bool {
		return expectedIds[i] < expectedIds[j]
	})
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	if diff := cmp.Diff(expectedIds, ids); diff != "" {
		t.Fatalf("unexpected job IDs: %s", diff)
	}
}

func BenchmarkJobStore_EnqueueJob_basic(b *testing.B) {
	ss, err := NewStateStore()
	if err != nil {
		b.Fatal(err)
	}

	tmpDir := b.TempDir()

	for i := 0; i < b.N; i++ {
		i := i
		dirPath := filepath.Join(tmpDir, fmt.Sprintf("folder-%d", i))

		_, err := ss.JobStore.EnqueueJob(job.Job{
			Func: func(c context.Context) error {
				return nil
			},
			Dir:  document.DirHandleFromPath(dirPath),
			Type: "test-type",
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	ids, err := ss.JobStore.ListQueuedJobs()
	if err != nil {
		b.Fatal(err)
	}

	if len(ids) != b.N {
		b.Fatalf("expected %d jobs, %d given", b.N, len(ids))
	}
}

func TestJobStore_EnqueueJob_verify(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss.SetLogger(testLogger())

	tmpDir := t.TempDir()

	jobCount := 50

	for i := 0; i < jobCount; i++ {
		i := i
		dirPath := filepath.Join(tmpDir, fmt.Sprintf("folder-%d", i))

		_, err := ss.JobStore.EnqueueJob(job.Job{
			Func: func(c context.Context) error {
				return nil
			},
			Dir:  document.DirHandleFromPath(dirPath),
			Type: "test-type",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	ids, err := ss.JobStore.ListQueuedJobs()
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != jobCount {
		t.Fatalf("expected %d jobs, %d given", jobCount, len(ids))
	}

	for _, id := range ids {
		err := ss.JobStore.FinishJob(id, nil)
		if err != nil {
			t.Error(err)
		}
	}

	ids, err = ss.JobStore.allJobs()
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 0 {
		t.Fatalf("expected %d jobs, %d given", 0, len(ids))
	}
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}

func TestJobStore_DequeueJobsForDir(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	firstDir := document.DirHandleFromPath("/test-1")
	_, err = ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  firstDir,
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  document.DirHandleFromPath("/test-2"),
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ss.JobStore.DequeueJobsForDir(firstDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedJobIds := job.IDs{id2}
	jobIds, err := ss.JobStore.allJobs()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectedJobIds, jobIds); diff != "" {
		t.Fatalf("unexpected jobs: %s", diff)
	}
}

func TestJobStore_AwaitNextJob_closedOnly(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	firstDir := document.DirHandleFromPath("/test-1")
	id1, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  firstDir,
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}

	secondDir := document.DirHandleFromPath("/test-2")
	_, err = ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  secondDir,
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ss.DocumentStore.OpenDocument(document.Handle{Dir: secondDir, Filename: "test.tf"}, "test", 0, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	nextId, job, err := ss.JobStore.AwaitNextJob(ctx, false)
	if err != nil {
		t.Fatal(err)
	}

	if nextId != id1 {
		t.Fatalf("expected next job ID %q, given: %q", id1, nextId)
	}

	if job.Dir != firstDir {
		t.Fatalf("expected next job dir %q, given: %q", firstDir, job.Dir)
	}

	if job.Type != "test-type" {
		t.Fatalf("expected next job dir %q, given: %q", "test-type", job.Type)
	}

	ctx, cancelFunc := context.WithTimeout(ctx, 250*time.Millisecond)
	t.Cleanup(cancelFunc)
	nextId, job, err = ss.JobStore.AwaitNextJob(ctx, false)
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("%#v", err)
		}
	}
}

func TestJobStore_AwaitNextJob_openOnly(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	firstDir := document.DirHandleFromPath("/test-1")
	_, err = ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  firstDir,
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}

	secondDir := document.DirHandleFromPath("/test-2")
	id2, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  secondDir,
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ss.DocumentStore.OpenDocument(document.Handle{Dir: secondDir, Filename: "test.tf"}, "test", 0, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	nextId, job, err := ss.JobStore.AwaitNextJob(ctx, true)
	if err != nil {
		t.Fatal(err)
	}

	if nextId != id2 {
		t.Fatalf("expected next job ID %q, given: %q", id2, nextId)
	}

	if job.Dir != secondDir {
		t.Fatalf("expected next job dir %q, given: %q", secondDir, job.Dir)
	}

	if job.Type != "test-type" {
		t.Fatalf("expected next job dir %q, given: %q", "test-type", job.Type)
	}

	ctx, cancelFunc := context.WithTimeout(ctx, 250*time.Millisecond)
	t.Cleanup(cancelFunc)
	nextId, job, err = ss.JobStore.AwaitNextJob(ctx, true)
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("%#v", err)
		}
	}
}

func TestJobStore_WaitForJobs(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	id1, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  document.DirHandleFromPath("/test-1"),
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}

	go func(jobStore *JobStore) {
		err := jobStore.FinishJob(id1, nil)
		if err != nil {
			log.Fatal(err)
		}
	}(ss.JobStore)

	ctx := context.Background()
	err = ss.JobStore.WaitForJobs(ctx, id1)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := ss.JobStore.ListQueuedJobs()
	if err != nil {
		t.Fatal(err)
	}

	expectedIds := job.IDs{}
	if diff := cmp.Diff(expectedIds, ids); diff != "" {
		t.Fatalf("unexpected jobs: %s", diff)
	}
}

func TestJobStore_FinishJob_basic(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	id1, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  document.DirHandleFromPath("/test-1"),
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  document.DirHandleFromPath("/test-2"),
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ss.JobStore.FinishJob(id1, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedQueuedIds := job.IDs{id2}
	queuedIds, err := ss.JobStore.ListQueuedJobs()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectedQueuedIds, queuedIds); diff != "" {
		t.Fatalf("unexpected queued jobs: %s", diff)
	}
}

func TestJobStore_FinishJob_defer(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	defer1Func := func(ctx context.Context, jobErr error) (ids job.IDs) {
		jobStore, err := job.JobStoreFromContext(ctx)
		if err != nil {
			return nil
		}

		id, err := jobStore.EnqueueJob(job.Job{
			Func: func(ctx context.Context) error {
				return nil
			},
			Dir:  document.DirHandleFromPath("/test-defer-1"),
			Type: "test-type",
		})
		if err != nil {
			return nil
		}
		ids = append(ids, id)
		return
	}

	id1, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:   document.DirHandleFromPath("/test-1"),
		Type:  "test-type",
		Defer: defer1Func,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx = job.WithJobStore(ctx, ss.JobStore)
	// execute deferred func, which is what scheduler would do
	deferredIds := defer1Func(ctx, nil)

	err = ss.JobStore.FinishJob(id1, nil, deferredIds...)
	if err != nil {
		t.Fatal(err)
	}

	expectedJobIds := job.IDs{id1, deferredIds[0]}
	jobIds, err := ss.JobStore.allJobs()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectedJobIds, jobIds); diff != "" {
		t.Fatalf("unexpected jobs: %s", diff)
	}

	err = ss.JobStore.FinishJob(deferredIds[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	expectedJobIds = job.IDs{}
	jobIds, err = ss.JobStore.allJobs()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectedJobIds, jobIds); diff != "" {
		t.Fatalf("unexpected jobs: %s", diff)
	}
}
