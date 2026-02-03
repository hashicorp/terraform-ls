// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
)

func TestChanges_dirOpenMark_openBeforeChange(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()
	modHandle := document.DirHandleFromPath(modPath)
	docHandle := document.Handle{
		Dir:      modHandle,
		Filename: "main.tf",
	}
	err = ss.DocumentStore.OpenDocument(docHandle, "terraform", 0, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	err = ss.ChangeStore.QueueChange(modHandle, Changes{})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	batch, err := ss.ChangeStore.AwaitNextChangeBatch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !batch.IsDirOpen {
		t.Fatalf("expected dir to be open for change batch, given: %#v", batch)
	}
}

func TestChanges_dirOpenMark_openAfterChange(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()
	modHandle := document.DirHandleFromPath(modPath)

	err = ss.ChangeStore.QueueChange(modHandle, Changes{})
	if err != nil {
		t.Fatal(err)
	}

	docHandle := document.Handle{
		Dir:      modHandle,
		Filename: "main.tf",
	}
	err = ss.DocumentStore.OpenDocument(docHandle, "terraform", 0, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	batch, err := ss.ChangeStore.AwaitNextChangeBatch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !batch.IsDirOpen {
		t.Fatalf("expected dir to be open for change batch, given: %#v", batch)
	}
}

func TestChanges_AwaitNextChangeBatch_maxTimespan(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()
	modHandle := document.DirHandleFromPath(modPath)

	ctx := context.Background()
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	_, err = ss.JobStore.EnqueueJob(ctx, job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  modHandle,
		Type: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ss.ChangeStore.QueueChange(modHandle, Changes{})
	if err != nil {
		t.Fatal(err)
	}

	// confirm the method gets cancelled with pending job
	// and less than maximum timespan to wait
	ctx, cancelFunc := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancelFunc()

	_, err = ss.ChangeStore.AwaitNextChangeBatch(ctx)
	if err == nil {
		t.Fatal("expected timeout")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context deadline exceeded error, given: %#v", err)
	}

}

func TestChanges_AwaitNextChangeBatch_multipleChanges(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	ss.ChangeStore.TimeProvider = testTimeProvider

	modHandle := document.DirHandleFromPath(t.TempDir())
	err = ss.ChangeStore.QueueChange(modHandle, Changes{})
	if err != nil {
		t.Fatal(err)
	}

	err = ss.ChangeStore.QueueChange(modHandle, Changes{
		TerraformVersion: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	batch, err := ss.ChangeStore.AwaitNextChangeBatch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expectedBatch := ChangeBatch{
		DirHandle:       modHandle,
		FirstChangeTime: testTimeProvider(),
		IsDirOpen:       false,
		Changes: Changes{
			TerraformVersion: true,
		},
	}
	if diff := cmp.Diff(expectedBatch, batch); diff != "" {
		t.Fatalf("unexpected change batch: %s", diff)
	}

	// verify that no more batches are available
	ctx, cancelFunc = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelFunc()
	_, err = ss.ChangeStore.AwaitNextChangeBatch(ctx)
	if err == nil {
		t.Fatal("expected error on next batch read")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context deadline exceeded error, given: %#v", err)
	}
}

func TestChanges_AwaitNextChangeBatch_removal(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	ss.ChangeStore.TimeProvider = testTimeProvider

	modHandle := document.DirHandleFromPath(t.TempDir())
	err = ss.ChangeStore.QueueChange(modHandle, Changes{})
	if err != nil {
		t.Fatal(err)
	}
	err = ss.ChangeStore.QueueChange(modHandle, Changes{
		IsRemoval: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	batch, err := ss.ChangeStore.AwaitNextChangeBatch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expectedBatch := ChangeBatch{
		DirHandle:       modHandle,
		FirstChangeTime: testTimeProvider(),
		IsDirOpen:       false,
		Changes: Changes{
			IsRemoval: true,
		},
	}
	if diff := cmp.Diff(expectedBatch, batch); diff != "" {
		t.Fatalf("unexpected change batch: %s", diff)
	}

	// verify that no more batches are available
	ctx, cancelFunc = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelFunc()
	_, err = ss.ChangeStore.AwaitNextChangeBatch(ctx)
	if err == nil {
		t.Fatal("expected error on next batch read")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context deadline exceeded error, given: %#v", err)
	}
}
