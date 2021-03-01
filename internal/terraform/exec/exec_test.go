package exec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-exec/tfinstall"
)

func TestExec_timeout(t *testing.T) {
	// This test is known to fail under '-race'
	// and similar race conditions are reproducible upstream
	// See https://github.com/hashicorp/terraform-exec/issues/129
	t.Skip("upstream implementation prone to race conditions")

	e := newExecutor(t)
	timeout := 1 * time.Millisecond
	e.SetTimeout(timeout)

	expectedErr := ExecTimeoutError("Version", timeout)

	_, _, err := e.Version(context.Background())
	if err != nil {
		if errors.Is(err, expectedErr) {
			return
		}

		t.Fatalf("errors don't match.\nexpected: %#v\ngiven:    %#v\n",
			expectedErr, err)
	}

	t.Fatalf("expected timeout error: %#v, given: %#v", expectedErr, err)
}

func TestExec_cancel(t *testing.T) {
	e := newExecutor(t)

	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()

	expectedErr := ExecCanceledError("Version")

	_, _, err := e.Version(ctx)
	if err != nil {
		if errors.Is(err, expectedErr) {
			return
		}

		t.Fatalf("errors don't match.\nexpected: %#v\ngiven:    %#v\n",
			expectedErr, err)
	}

	t.Fatalf("expected cancel error: %#v, given: %#v", expectedErr, err)
}

func newExecutor(t *testing.T) TerraformExecutor {
	tmpDir := TempDir(t)
	workDir := filepath.Join(tmpDir, "workdir")
	err := os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Join(tmpDir, "installdir")
	err = os.MkdirAll(installDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	opts := tfinstall.ExactVersion("0.13.1", installDir)
	execPath, err := opts.ExecPath(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	e, err := NewExecutor(workDir, execPath)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TempDir(t *testing.T) string {
	tmpDir := filepath.Join(os.TempDir(), "terraform-ls", t.Name())

	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
	})
	return tmpDir
}
