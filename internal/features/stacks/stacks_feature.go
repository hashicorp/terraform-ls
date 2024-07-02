// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package stacks

import (
	"context"
	"io"
	"log"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/features/modules/jobs"
	stackDecoder "github.com/hashicorp/terraform-ls/internal/features/stacks/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
)

type StacksFeature struct {
	store      *state.StackStore
	stateStore *globalState.StateStore
	bus        *eventbus.EventBus
	fs         jobs.ReadOnlyFS
	logger     *log.Logger
	stopFunc   context.CancelFunc
}

func NewStacksFeature(bus *eventbus.EventBus, stateStore *globalState.StateStore, fs jobs.ReadOnlyFS) (*StacksFeature, error) {
	store, err := state.NewStackStore()
	if err != nil {
		return nil, err
	}
	discardLogger := log.New(io.Discard, "", 0)

	return &StacksFeature{
		store:      store,
		bus:        bus,
		fs:         fs,
		stateStore: stateStore,
		logger:     discardLogger,
		stopFunc:   func() {},
	}, nil
}

func (f *StacksFeature) SetLogger(logger *log.Logger) {
	f.logger = logger
	f.store.SetLogger(logger)
}

// Start starts the features separate goroutine.
// It listens to various events from the EventBus and performs corresponding actions.
func (f *StacksFeature) Start(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	f.stopFunc = cancelFunc

	topic := "feature.stacks"

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

func (f *StacksFeature) Stop() {
	f.stopFunc()
	f.logger.Print("stopped stacks feature")
}

func (f *StacksFeature) PathContext(path lang.Path) (*decoder.PathContext, error) {
	pathReader := &stackDecoder.PathReader{
		StateReader: f.store,
	}

	return pathReader.PathContext(path)
}

func (f *StacksFeature) Paths(ctx context.Context) []lang.Path {
	pathReader := &stackDecoder.PathReader{
		StateReader: f.store,
	}

	return pathReader.Paths(ctx)
}

func (f *StacksFeature) Diagnostics(path string) diagnostics.Diagnostics {
	// TODO: This is not hooked up to the change diagnostics system yet
	diags := diagnostics.NewDiagnostics()

	mod, err := f.store.StackRecordByPath(path)
	if err != nil {
		return diags
	}

	for source, dm := range mod.StackDiagnostics {
		diags.Append(source, dm.AutoloadedOnly().AsMap())
	}

	for source, dm := range mod.DeployDiagnostics {
		diags.Append(source, dm.AutoloadedOnly().AsMap())
	}

	return diags
}
