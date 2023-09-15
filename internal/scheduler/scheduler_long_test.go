// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build longtest

package scheduler

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/state"
)

// See https://github.com/hashicorp/terraform-ls/issues/1065
// This test can be very expensive to run in terms of CPU, memory and time.
// It takes about 3-4 minutes to finish on M1 Pro.
func TestScheduler_millionJobsQueued(t *testing.T) {
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss.SetLogger(testLogger())

	tmpDir := t.TempDir()
	ctx, cancelFunc := context.WithCancel(context.Background())
	ctx = lsctx.WithRPCContext(ctx, lsctx.RPCContextData{})

	lowPrioSched := NewScheduler(ss.JobStore, 1, job.LowPriority)
	lowPrioSched.Start(ctx)
	t.Cleanup(func() {
		lowPrioSched.Stop()
		cancelFunc()
	})

	highPrioSched := NewScheduler(ss.JobStore, 100, job.HighPriority)

	// slightly over ~1M jobs seems sufficient to exceed the goroutine stack limit
	idBatches := make([]job.IDs, 106, 106)
	var wg sync.WaitGroup
	for i := 0; i <= 105; i++ {
		wg.Add(1)
		i := i
		go func(i int) {
			defer wg.Done()
			idBatches[i] = make(job.IDs, 0)
			for j := 0; j < 10000; j++ {
				dirPath := filepath.Join(tmpDir, fmt.Sprintf("folder-%d", j))

				newId, err := ss.JobStore.EnqueueJob(ctx, job.Job{
					Func: func(c context.Context) error {
						return nil
					},
					Dir:      document.DirHandleFromPath(dirPath),
					Type:     "test",
					Priority: job.HighPriority,
				})
				if err != nil {
					t.Error(err)
				}
				idBatches[i] = append(idBatches[i], newId)
			}
			t.Logf("scheduled %d high priority jobs in batch %d", len(idBatches[i]), i)
		}(i)
	}
	wg.Wait()

	highPrioSched.Start(ctx)
	t.Log("high priority scheduler started")

	t.Cleanup(func() {
		highPrioSched.Stop()
		cancelFunc()
	})

	for _, batch := range idBatches {
		err = ss.JobStore.WaitForJobs(ctx, batch...)
		if err != nil {
			t.Fatal(err)
		}
	}
}
