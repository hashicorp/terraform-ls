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
	modPath := t.TempDir()
	notifier := NewNotifier(mockModuleStore{modPath: modPath}, mockVarsStore{modPath: modPath}, []Hook{
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

type mockVarsStore struct {
	modPath string
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

func (mms mockModuleStore) ModuleByPath(path string) (*state.Module, error) {
	if path != mms.modPath {
		return nil, fmt.Errorf("unexpected path: %q", path)
	}

	return &state.Module{
		Path: path,
	}, nil
}

func (mvs mockVarsStore) VarsByPath(path string) (*state.Vars, error) {
	if path != mvs.modPath {
		return nil, fmt.Errorf("unexpected path: %q", path)
	}

	return &state.Vars{
		Path: path,
	}, nil
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.Default()
	}
	return log.New(ioutil.Discard, "", 0)
}
