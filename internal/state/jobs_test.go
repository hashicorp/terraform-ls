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

func TestJobStore_EnqueueJob_openDir(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	dirHandle := document.DirHandleFromPath("/test-1")

	err = ss.DocumentStore.OpenDocument(document.Handle{Dir: dirHandle, Filename: "test.tf"}, "test", 0, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	id, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  dirHandle,
		Type: "test-type",
	})
	if err != nil {
		t.Fatal(err)
	}

	// verify that job for open dir comes is treated as high priority
	ctx := context.Background()
	ctx, cancelFunc := context.WithTimeout(ctx, 250*time.Millisecond)
	t.Cleanup(cancelFunc)
	nextId, j, err := ss.JobStore.AwaitNextJob(ctx, job.HighPriority)
	if err != nil {
		t.Fatal(err)
	}

	if nextId != id {
		t.Fatalf("expected next job ID %q, given: %q", id, nextId)
	}

	if j.Dir != dirHandle {
		t.Fatalf("expected next job dir %q, given: %q", dirHandle, j.Dir)
	}

	if j.Type != "test-type" {
		t.Fatalf("expected next job dir %q, given: %q", "test-type", j.Type)
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
	nextId, j, err := ss.JobStore.AwaitNextJob(ctx, job.LowPriority)
	if err != nil {
		t.Fatal(err)
	}

	if nextId != id1 {
		t.Fatalf("expected next job ID %q, given: %q", id1, nextId)
	}

	if j.Dir != firstDir {
		t.Fatalf("expected next job dir %q, given: %q", firstDir, j.Dir)
	}

	if j.Type != "test-type" {
		t.Fatalf("expected next job dir %q, given: %q", "test-type", j.Type)
	}

	ctx, cancelFunc := context.WithTimeout(ctx, 250*time.Millisecond)
	t.Cleanup(cancelFunc)
	nextId, j, err = ss.JobStore.AwaitNextJob(ctx, job.LowPriority)
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
	nextId, j, err := ss.JobStore.AwaitNextJob(ctx, job.HighPriority)
	if err != nil {
		t.Fatal(err)
	}

	if nextId != id2 {
		t.Fatalf("expected next job ID %q, given: %q", id2, nextId)
	}

	if j.Dir != secondDir {
		t.Fatalf("expected next job dir %q, given: %q", secondDir, j.Dir)
	}

	if j.Type != "test-type" {
		t.Fatalf("expected next job dir %q, given: %q", "test-type", j.Type)
	}

	ctx, cancelFunc := context.WithTimeout(ctx, 250*time.Millisecond)
	t.Cleanup(cancelFunc)
	nextId, j, err = ss.JobStore.AwaitNextJob(ctx, job.HighPriority)
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("%#v", err)
		}
	}
}

func TestJobStore_AwaitNextJob_highPriority(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	firstDir := document.DirHandleFromPath("/test-1")
	id1, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:      firstDir,
		Type:     "test-type",
		Priority: job.HighPriority,
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
	nextId, j, err := ss.JobStore.AwaitNextJob(ctx, job.HighPriority)
	if err != nil {
		t.Fatal(err)
	}

	if nextId != id1 {
		t.Fatalf("expected next job ID %q, given: %q", id1, nextId)
	}

	if j.Dir != firstDir {
		t.Fatalf("expected next job dir %q, given: %q", firstDir, j.Dir)
	}

	if j.Type != "test-type" {
		t.Fatalf("expected next job dir %q, given: %q", "test-type", j.Type)
	}

	nextId, j, err = ss.JobStore.AwaitNextJob(ctx, job.HighPriority)
	if err != nil {
		t.Fatal(err)
	}

	if nextId != id2 {
		t.Fatalf("expected next job ID %q, given: %q", id2, nextId)
	}

	if j.Dir != secondDir {
		t.Fatalf("expected next job dir %q, given: %q", secondDir, j.Dir)
	}

	if j.Type != "test-type" {
		t.Fatalf("expected next job dir %q, given: %q", "test-type", j.Type)
	}

	ctx, cancelFunc := context.WithTimeout(ctx, 250*time.Millisecond)
	t.Cleanup(cancelFunc)
	nextId, j, err = ss.JobStore.AwaitNextJob(ctx, job.HighPriority)
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("%#v", err)
		}
	}
}

func TestJobStore_AwaitNextJob_lowPriority(t *testing.T) {
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
	id2, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:      secondDir,
		Type:     "test-type",
		Priority: job.LowPriority,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ss.DocumentStore.OpenDocument(document.Handle{Dir: secondDir, Filename: "test.tf"}, "test", 0, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	baseCtx := context.Background()

	ctx, cancelFunc := context.WithTimeout(baseCtx, 250*time.Millisecond)
	t.Cleanup(cancelFunc)
	_, _, err = ss.JobStore.AwaitNextJob(ctx, job.HighPriority)
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("%#v", err)
		}
	} else {
		t.Fatal("expected error")
	}

	nextId, j, err := ss.JobStore.AwaitNextJob(baseCtx, job.LowPriority)
	if err != nil {
		t.Fatal(err)
	}

	if nextId != id1 {
		t.Fatalf("expected next job ID %q, given: %q", id1, nextId)
	}

	if j.Dir != firstDir {
		t.Fatalf("expected next job dir %q, given: %q", firstDir, j.Dir)
	}

	if j.Type != "test-type" {
		t.Fatalf("expected next job dir %q, given: %q", "test-type", j.Type)
	}

	nextId, j, err = ss.JobStore.AwaitNextJob(baseCtx, job.LowPriority)
	if err != nil {
		t.Fatal(err)
	}

	if nextId != id2 {
		t.Fatalf("expected next job ID %q, given: %q", id2, nextId)
	}

	if j.Dir != secondDir {
		t.Fatalf("expected next job dir %q, given: %q", secondDir, j.Dir)
	}

	if j.Type != "test-type" {
		t.Fatalf("expected next job dir %q, given: %q", "test-type", j.Type)
	}

	ctx, cancelFunc = context.WithTimeout(baseCtx, 250*time.Millisecond)
	t.Cleanup(cancelFunc)
	nextId, j, err = ss.JobStore.AwaitNextJob(ctx, job.HighPriority)
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("%#v", err)
		}
	} else {
		t.Fatal("expected error")
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

	defer1Func := func(ctx context.Context, jobErr error) (job.IDs, error) {
		ids := make(job.IDs, 0)
		jobStore := ss.JobStore

		id, err := jobStore.EnqueueJob(job.Job{
			Func: func(ctx context.Context) error {
				return nil
			},
			Dir:  document.DirHandleFromPath("/test-defer-1"),
			Type: "test-type",
		})
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
		return ids, err
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
	// execute deferred func, which is what scheduler would do
	deferredIds, err := defer1Func(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

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

func TestJobStore_FinishJob_dependsOn(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	parentId, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  document.DirHandleFromPath(t.TempDir()),
		Type: "parent-job",
	})
	if err != nil {
		t.Fatal(err)
	}

	childId, err := ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:       document.DirHandleFromPath(t.TempDir()),
		Type:      "child-job",
		DependsOn: job.IDs{parentId},
	})
	if err != nil {
		t.Fatal(err)
	}

	ids, err := ss.JobStore.ListQueuedJobs()
	if err != nil {
		t.Fatal(err)
	}
	expectedIds := job.IDs{parentId, childId}
	if diff := cmp.Diff(expectedIds, ids); diff != "" {
		t.Fatalf("unexpected IDs: %s", diff)
	}

	err = ss.JobStore.FinishJob(parentId, nil)
	if err != nil {
		t.Fatal(err)
	}

	ids, err = ss.JobStore.ListQueuedJobs()
	if err != nil {
		t.Fatal(err)
	}
	expectedIds = job.IDs{childId}
	if diff := cmp.Diff(expectedIds, ids); diff != "" {
		t.Fatalf("unexpected IDs after finishing: %s", diff)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	t.Cleanup(cancelFunc)
	nextId, j, err := ss.JobStore.AwaitNextJob(ctx, job.LowPriority)
	if err != nil {
		t.Fatal(err)
	}
	if nextId != childId {
		t.Fatalf("expected next ID %q, given %q", childId, nextId)
	}
	expectedDependsOn := job.IDs{}
	if diff := cmp.Diff(expectedDependsOn, j.DependsOn); diff != "" {
		t.Fatalf("unexpected DependsOn: %s", diff)
	}
}
