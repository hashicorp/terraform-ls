// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package notifier

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/state"
)

func TestNotifier(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(2)

	hookFunc := func(ctx context.Context, changes state.ModuleChanges) error {
		wg.Done()
		cancelFunc()
		return nil
	}
	notifier := NewNotifier(mockModuleStore{modPath: t.TempDir()}, []Hook{
		hookFunc,
		hookFunc,
	})
	notifier.SetLogger(testLogger())

	notifier.Start(ctx)

	wg.Wait()
}

type mockModuleStore struct {
	returned bool
	modPath  string
}

func (mms mockModuleStore) AwaitNextChangeBatch(ctx context.Context) (state.ModuleChangeBatch, error) {
	if mms.returned {
		return state.ModuleChangeBatch{}, fmt.Errorf("no more batches")
	}
	defer func() { mms.returned = true }()

	return state.ModuleChangeBatch{
		DirHandle:       document.DirHandleFromPath(mms.modPath),
		FirstChangeTime: time.Date(2022, 5, 26, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (mms mockModuleStore) ModuleByPath(path string) (*state.ModuleRecord, error) {
	if path != mms.modPath {
		return nil, fmt.Errorf("unexpected path: %q", path)
	}

	return state.NewModuleTest(path), nil
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.Default()
	}
	return log.New(ioutil.Discard, "", 0)
}
