package state

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-ls/internal/document"
)

func TestWalkerPathStore_EnqueueDir(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	dirHandle := document.DirHandleFromPath(tmpDir)

	err = ss.WalkerPaths.EnqueueDir(dirHandle)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	wp, err := ss.WalkerPaths.AwaitNextDir(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if wp == nil {
		t.Fatalf("expected next dir: %q, nil given", dirHandle)
	}
	if wp.Dir != dirHandle {
		t.Fatalf("expected next dir: %q\ngiven next dir: %q", dirHandle, wp.Dir)
	}
}

func TestWalkerPathStore_DequeueDir_queued(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	alphaHandle := document.DirHandleFromPath(filepath.Join(tmpDir, "alpha"))
	err = ss.WalkerPaths.EnqueueDir(alphaHandle)
	if err != nil {
		t.Fatal(err)
	}
	betaHandle := document.DirHandleFromPath(filepath.Join(tmpDir, "beta"))
	err = ss.WalkerPaths.EnqueueDir(betaHandle)
	if err != nil {
		t.Fatal(err)
	}

	err = ss.WalkerPaths.DequeueDir(alphaHandle)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelFunc()

	wp, err := ss.WalkerPaths.AwaitNextDir(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if wp == nil {
		t.Fatalf("expected next dir: %q, nil given", betaHandle)
	}
	if wp.Dir != betaHandle {
		t.Fatalf("expected next dir: %q\ngiven next dir: %q", betaHandle, wp.Dir)
	}

	_, err = ss.WalkerPaths.AwaitNextDir(ctx, false)
	if err != nil {
		if err == context.DeadlineExceeded {
			// expected timeout
			return
		}
		t.Fatal(err)
	}
	t.Fatal("expected error for next dir")
}

func TestWalkerPathStore_DequeueDir_notQueued(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	dirHandle := document.DirHandleFromPath(tmpDir)
	err = ss.WalkerPaths.EnqueueDir(dirHandle)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelFunc()

	_, err = ss.WalkerPaths.AwaitNextDir(ctx, false)
	if err != nil {
		t.Fatal(err)
	}

	err = ss.WalkerPaths.DequeueDir(dirHandle)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWalkerPathStore_RemoveDir(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	alphaHandle := document.DirHandleFromPath(filepath.Join(tmpDir, "alpha"))
	err = ss.WalkerPaths.EnqueueDir(alphaHandle)
	if err != nil {
		t.Fatal(err)
	}
	betaHandle := document.DirHandleFromPath(filepath.Join(tmpDir, "beta"))
	err = ss.WalkerPaths.EnqueueDir(betaHandle)
	if err != nil {
		t.Fatal(err)
	}

	err = ss.WalkerPaths.RemoveDir(alphaHandle)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	wp, err := ss.WalkerPaths.AwaitNextDir(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if wp == nil {
		t.Fatalf("expected next dir: %q, nil given", betaHandle)
	}
	if wp.Dir != betaHandle {
		t.Fatalf("expected next dir: %q\ngiven next dir: %q", betaHandle, wp.Dir)
	}

	err = ss.WalkerPaths.RemoveDir(betaHandle)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelFunc := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancelFunc()

	_, err = ss.WalkerPaths.AwaitNextDir(ctx, false)
	if err != nil {
		if err == context.DeadlineExceeded {
			// expected timeout
			return
		}
		t.Fatal(err)
	}
	t.Fatal("expected error for next dir")
}

func TestWalkerPathStore_AwaitNextDir_openOnly(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	alphaHandle := document.DirHandleFromPath(filepath.Join(tmpDir, "alpha"))
	err = ss.WalkerPaths.EnqueueDir(alphaHandle)
	if err != nil {
		t.Fatal(err)
	}
	dh := document.HandleFromPath(filepath.Join(tmpDir, "alpha", "test.tf"))
	err = ss.DocumentStore.OpenDocument(dh, "", 0, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	betaHandle := document.DirHandleFromPath(filepath.Join(tmpDir, "beta"))
	err = ss.WalkerPaths.EnqueueDir(betaHandle)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	wp, err := ss.WalkerPaths.AwaitNextDir(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if wp == nil {
		t.Fatalf("expected next dir: %q, nil given", alphaHandle)
	}
	if wp.Dir != alphaHandle {
		t.Fatalf("expected next dir: %q\ngiven next dir: %q", alphaHandle, wp.Dir)
	}

	ctx, cancelFunc := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancelFunc()

	_, err = ss.WalkerPaths.AwaitNextDir(ctx, true)
	if err != nil {
		if err == context.DeadlineExceeded {
			// expected timeout
			return
		}
		t.Fatal(err)
	}
	t.Fatal("expected error for next dir")
}

func TestWalkerPathStore_WaitForDirs(t *testing.T) {
	ss, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	alphaHandle := document.DirHandleFromPath(filepath.Join(tmpDir, "alpha"))
	err = ss.WalkerPaths.EnqueueDir(alphaHandle)
	if err != nil {
		t.Fatal(err)
	}
	betaHandle := document.DirHandleFromPath(filepath.Join(tmpDir, "beta"))
	err = ss.WalkerPaths.EnqueueDir(betaHandle)
	if err != nil {
		t.Fatal(err)
	}

	go func(t *testing.T) {
		ctx := context.Background()
		_, err := ss.WalkerPaths.AwaitNextDir(ctx, false)
		if err != nil {
			t.Error(err)
		}
		err = ss.WalkerPaths.RemoveDir(alphaHandle)
		if err != nil {
			t.Error(err)
		}
	}(t)
	go func(t *testing.T) {
		err := ss.WalkerPaths.RemoveDir(betaHandle)
		if err != nil {
			t.Error(err)
		}
	}(t)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Millisecond)
	t.Cleanup(cancelFunc)

	err = ss.WalkerPaths.WaitForDirs(ctx, []document.DirHandle{
		alphaHandle,
		betaHandle,
	})
	if err != nil {
		t.Fatal(err)
	}
}
