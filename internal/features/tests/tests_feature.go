// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"context"
	"io"
	"log"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/features/modules/jobs"
	testDecoder "github.com/hashicorp/terraform-ls/internal/features/tests/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/tests/state"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
)

type TestsFeature struct {
	store         *state.TestStore
	stateStore    *globalState.StateStore
	bus           *eventbus.EventBus
	fs            jobs.ReadOnlyFS
	logger        *log.Logger
	stopFunc      context.CancelFunc
	moduleFeature testDecoder.ModuleReader
	rootFeature   testDecoder.RootReader
}

func NewTestsFeature(bus *eventbus.EventBus, stateStore *globalState.StateStore, fs jobs.ReadOnlyFS, moduleFeature testDecoder.ModuleReader, rootFeature testDecoder.RootReader) (*TestsFeature, error) {
	store, err := state.NewTestStore(stateStore.ChangeStore, stateStore.ProviderSchemas)
	if err != nil {
		return nil, err
	}
	discardLogger := log.New(io.Discard, "", 0)

	return &TestsFeature{
		store:         store,
		bus:           bus,
		fs:            fs,
		stateStore:    stateStore,
		moduleFeature: moduleFeature,
		rootFeature:   rootFeature,
		logger:        discardLogger,
		stopFunc:      func() {},
	}, nil
}

func (f *TestsFeature) SetLogger(logger *log.Logger) {
	f.logger = logger
	f.store.SetLogger(logger)
}

// Start starts the features separate goroutine.
// It listens to various events from the EventBus and performs corresponding actions.
func (f *TestsFeature) Start(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	f.stopFunc = cancelFunc

	topic := "feature.tests"

	didOpenDone := make(chan struct{}, 10)
	didChangeDone := make(chan struct{}, 10)
	didChangeWatchedDone := make(chan struct{}, 10)

	discover := f.bus.OnDiscover(topic, nil)
	didOpen := f.bus.OnDidOpen(topic, didOpenDone)
	didChange := f.bus.OnDidChange(topic, didChangeDone)
	didChangeWatched := f.bus.OnDidChangeWatched(topic, didChangeWatchedDone)

	go func() {
		for {
			select {
			case discover := <-discover:
				// TODO? collect errors
				f.discover(discover.Path, discover.Files)
			case didOpen := <-didOpen:
				// TODO? collect errors
				f.didOpen(didOpen.Context, didOpen.Dir, didOpen.LanguageID)
				didOpenDone <- struct{}{}
			case didChange := <-didChange:
				// TODO? collect errors
				f.didChange(didChange.Context, didChange.Dir)
				didChangeDone <- struct{}{}
			case didChangeWatched := <-didChangeWatched:
				// TODO? collect errors
				f.didChangeWatched(didChangeWatched.Context, didChangeWatched.RawPath, didChangeWatched.ChangeType, didChangeWatched.IsDir)
				didChangeWatchedDone <- struct{}{}

			case <-ctx.Done():
				return
			}
		}
	}()
}

func (f *TestsFeature) Stop() {
	f.stopFunc()
	f.logger.Print("stopped tests feature")
}

func (f *TestsFeature) PathContext(path lang.Path) (*decoder.PathContext, error) {
	pathReader := &testDecoder.PathReader{
		StateReader:  f.store,
		ModuleReader: f.moduleFeature,
		RootReader:   f.rootFeature,
	}

	return pathReader.PathContext(path)
}

func (f *TestsFeature) Paths(ctx context.Context) []lang.Path {
	pathReader := &testDecoder.PathReader{
		StateReader:  f.store,
		ModuleReader: f.moduleFeature,
		RootReader:   f.rootFeature,
	}

	return pathReader.Paths(ctx)
}

func (f *TestsFeature) Diagnostics(path string) diagnostics.Diagnostics {
	diags := diagnostics.NewDiagnostics()

	mod, err := f.store.TestRecordByPath(path)
	if err != nil {
		return diags
	}

	for source, dm := range mod.Diagnostics {
		diags.Append(source, dm.AutoloadedOnly().AsMap())
	}

	return diags
}
