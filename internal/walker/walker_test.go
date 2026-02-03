// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package walker

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/state"
)

func TestWalker_basic(t *testing.T) {
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	pa := state.NewPathAwaiter(ss.WalkerPaths, false)
	bus := eventbus.NewEventBus()

	w := NewWalker(fs, pa, bus)
	w.Collector = NewWalkerCollector()
	w.SetLogger(testLogger())

	root, err := filepath.Abs(filepath.Join("testdata", "uninitialized-root"))
	if err != nil {
		t.Fatal(err)
	}
	dir := document.DirHandleFromPath(root)

	ctx := context.Background()
	err = ss.WalkerPaths.EnqueueDir(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = w.StartWalking(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = ss.WalkerPaths.WaitForDirs(ctx, []document.DirHandle{dir})
	if err != nil {
		t.Fatal(err)
	}
	err = ss.JobStore.WaitForJobs(ctx, w.Collector.JobIds()...)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Collector.ErrorOrNil()
	if err != nil {
		t.Fatal(err)
	}
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(io.Discard, "", 0)
}
