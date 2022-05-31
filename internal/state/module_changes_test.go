package state

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

func TestModuleChanges_dirOpenMark_openBeforeChange(t *testing.T) {
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

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	batch, err := ss.Modules.AwaitNextChangeBatch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !batch.IsDirOpen {
		t.Fatalf("expected dir to be open for change batch, given: %#v", batch)
	}
}

func TestModuleChanges_dirOpenMark_openAfterChange(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	modHandle := document.DirHandleFromPath(modPath)
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

	batch, err := ss.Modules.AwaitNextChangeBatch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !batch.IsDirOpen {
		t.Fatalf("expected dir to be open for change batch, given: %#v", batch)
	}
}

func TestModuleChanges_AwaitNextChangeBatch_maxTimespan(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()
	modHandle := document.DirHandleFromPath(modPath)

	_, err = ss.JobStore.EnqueueJob(job.Job{
		Func: func(ctx context.Context) error {
			return nil
		},
		Dir:  modHandle,
		Type: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	// confirm the method gets cancelled with pending job
	// and less than maximum timespan to wait
	ctx, cancelFunc := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelFunc()

	_, err = ss.Modules.AwaitNextChangeBatch(ctx)
	if err == nil {
		t.Fatal("expected timeout")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context deadline exceeded error, given: %#v", err)
	}

}

func TestModuleChanges_AwaitNextChangeBatch_multipleChanges(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	ss.Modules.TimeProvider = testTimeProvider

	modPath := t.TempDir()

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = ss.Modules.UpdateTerraformVersion(modPath, testVersion(t, "1.0.0"), map[tfaddr.Provider]*version.Version{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	batch, err := ss.Modules.AwaitNextChangeBatch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expectedBatch := ModuleChangeBatch{
		DirHandle:       document.DirHandleFromPath(modPath),
		FirstChangeTime: testTimeProvider(),
		IsDirOpen:       false,
		Changes: ModuleChanges{
			TerraformVersion: true,
		},
	}
	if diff := cmp.Diff(expectedBatch, batch); diff != "" {
		t.Fatalf("unexpected change batch: %s", diff)
	}

	// verify that no more batches are available
	ctx, cancelFunc = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelFunc()
	_, err = ss.Modules.AwaitNextChangeBatch(ctx)
	if err == nil {
		t.Fatal("expected error on next batch read")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context deadline exceeded error, given: %#v", err)
	}
}

func TestModuleChanges_AwaitNextChangeBatch_removal(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	ss.Modules.TimeProvider = testTimeProvider

	modPath := t.TempDir()

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = ss.Modules.Remove(modPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	batch, err := ss.Modules.AwaitNextChangeBatch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expectedBatch := ModuleChangeBatch{
		DirHandle:       document.DirHandleFromPath(modPath),
		FirstChangeTime: testTimeProvider(),
		IsDirOpen:       false,
		Changes: ModuleChanges{
			IsRemoval: true,
		},
	}
	if diff := cmp.Diff(expectedBatch, batch); diff != "" {
		t.Fatalf("unexpected change batch: %s", diff)
	}

	// verify that no more batches are available
	ctx, cancelFunc = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelFunc()
	_, err = ss.Modules.AwaitNextChangeBatch(ctx)
	if err == nil {
		t.Fatal("expected error on next batch read")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context deadline exceeded error, given: %#v", err)
	}
}
