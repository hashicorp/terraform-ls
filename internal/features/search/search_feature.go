// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package search

import (
	"context"
	"io"
	"log"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/features/modules/jobs"
	searchDecoder "github.com/hashicorp/terraform-ls/internal/features/search/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/search/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
)

type SearchFeature struct {
	store      *state.SearchStore
	stateStore *globalState.StateStore
	bus        *eventbus.EventBus
	fs         jobs.ReadOnlyFS
	logger     *log.Logger
	stopFunc   context.CancelFunc

	moduleFeature searchDecoder.ModuleReader
	rootFeature   searchDecoder.RootReader
}

func NewSearchFeature(bus *eventbus.EventBus, stateStore *globalState.StateStore, fs jobs.ReadOnlyFS, moduleFeature searchDecoder.ModuleReader, rootFeature searchDecoder.RootReader) (*SearchFeature, error) {
	store, err := state.NewSearchStore(stateStore.ChangeStore, stateStore.ProviderSchemas)
	if err != nil {
		return nil, err
	}
	discardLogger := log.New(io.Discard, "", 0)

	return &SearchFeature{
		store:         store,
		bus:           bus,
		fs:            fs,
		stateStore:    stateStore,
		logger:        discardLogger,
		stopFunc:      func() {},
		moduleFeature: moduleFeature,
		rootFeature:   rootFeature,
	}, nil
}

func (f *SearchFeature) SetLogger(logger *log.Logger) {
	f.logger = logger
	f.store.SetLogger(logger)
}

// Start starts the features separate goroutine.
// It listens to various events from the EventBus and performs corresponding actions.
func (f *SearchFeature) Start(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	f.stopFunc = cancelFunc

	topic := "feature.search"

	didOpenDone := make(chan job.IDs, 10)
	didChangeDone := make(chan job.IDs, 10)
	didChangeWatchedDone := make(chan job.IDs, 10)

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
				spawnedIds, _ := f.didOpen(didOpen.Context, didOpen.Dir, didOpen.LanguageID)
				didOpenDone <- spawnedIds
			case didChange := <-didChange:
				// TODO? collect errors
				spawnedIds, _ := f.didChange(didChange.Context, didChange.Dir)
				didChangeDone <- spawnedIds
			case didChangeWatched := <-didChangeWatched:
				// TODO? collect errors
				spawnedIds, _ := f.didChangeWatched(didChangeWatched.Context, didChangeWatched.RawPath, didChangeWatched.ChangeType, didChangeWatched.IsDir)
				didChangeWatchedDone <- spawnedIds

			case <-ctx.Done():
				return
			}
		}
	}()
}

func (f *SearchFeature) Stop() {
	f.stopFunc()
	f.logger.Print("stopped search feature")
}

func (f *SearchFeature) PathContext(path lang.Path) (*decoder.PathContext, error) {
	pathReader := &searchDecoder.PathReader{
		StateReader:  f.store,
		ModuleReader: f.moduleFeature,
		RootReader:   f.rootFeature,
	}

	return pathReader.PathContext(path)
}

func (f *SearchFeature) Paths(ctx context.Context) []lang.Path {
	pathReader := &searchDecoder.PathReader{
		StateReader:  f.store,
		ModuleReader: f.moduleFeature,
		RootReader:   f.rootFeature,
	}

	return pathReader.Paths(ctx)
}

func (f *SearchFeature) Diagnostics(path string) diagnostics.Diagnostics {
	diags := diagnostics.NewDiagnostics()

	mod, err := f.store.GetSearchRecordByPath(path)
	if err != nil {
		return diags
	}

	for source, dm := range mod.Diagnostics {
		diags.Append(source, dm.AutoloadedOnly().AsMap())
	}

	return diags
}

func (f *SearchFeature) Telemetry(path string) map[string]interface{} {
	properties := make(map[string]interface{})
	properties["search"] = true
	return properties
}
