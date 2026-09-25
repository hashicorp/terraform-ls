// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package notifier

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
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

	hookFunc := func(ctx context.Context, changes state.Changes) error {
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

func TestNotifier_cancellationIsNotLoggedAsFailure(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	logBuf := &bytes.Buffer{}
	var mu sync.Mutex
	safeWriter := writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return logBuf.Write(p)
	})

	awaiting := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)
	notifier := NewNotifier(cancellingChangeStore{
		awaiting: awaiting,
		stopped:  stopped,
	}, []Hook{})
	notifier.SetLogger(log.New(safeWriter, "", 0))

	notifier.Start(ctx)

	select {
	case <-awaiting:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the notifier to await a change batch")
	}
	cancelFunc()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the notifier to stop")
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	logged := logBuf.String()
	mu.Unlock()

	if strings.Contains(logged, "failed to notify") {
		t.Fatalf("shutdown reported as a failure: %s", logged)
	}
}

type cancellingChangeStore struct {
	awaiting chan struct{}
	stopped  chan struct{}
}

func (cs cancellingChangeStore) AwaitNextChangeBatch(ctx context.Context) (state.ChangeBatch, error) {
	signal(cs.awaiting)
	<-ctx.Done()
	signal(cs.stopped)
	return state.ChangeBatch{}, ctx.Err()
}

func signal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

type writerFunc func(p []byte) (int, error)

func (w writerFunc) Write(p []byte) (int, error) { return w(p) }

type mockModuleStore struct {
	returned bool
	modPath  string
}

func (mms mockModuleStore) AwaitNextChangeBatch(ctx context.Context) (state.ChangeBatch, error) {
	if mms.returned {
		return state.ChangeBatch{}, fmt.Errorf("no more batches")
	}
	defer func() { mms.returned = true }()

	return state.ChangeBatch{
		DirHandle:       document.DirHandleFromPath(mms.modPath),
		FirstChangeTime: time.Date(2022, 5, 26, 0, 0, 0, 0, time.UTC),
	}, nil
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.Default()
	}
	return log.New(ioutil.Discard, "", 0)
}
