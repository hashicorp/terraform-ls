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
	"github.com/hashicorp/terraform-ls/internal/job"
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

	moduleFeature stackDecoder.ModuleReader
	rootFeature   stackDecoder.RootReader
}

func NewStacksFeature(bus *eventbus.EventBus, stateStore *globalState.StateStore, fs jobs.ReadOnlyFS, moduleFeature stackDecoder.ModuleReader, rootFeature stackDecoder.RootReader) (*StacksFeature, error) {
	store, err := state.NewStackStore(stateStore.ChangeStore, stateStore.ProviderSchemas)
	if err != nil {
		return nil, err
	}
	discardLogger := log.New(io.Discard, "", 0)

	return &StacksFeature{
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

func (f *StacksFeature) Stop() {
	f.stopFunc()
	f.logger.Print("stopped stacks feature")
}

func (f *StacksFeature) PathContext(path lang.Path) (*decoder.PathContext, error) {
	pathReader := &stackDecoder.PathReader{
		StateReader:  f.store,
		ModuleReader: f.moduleFeature,
		RootReader:   f.rootFeature,
	}

	return pathReader.PathContext(path)
}

func (f *StacksFeature) Paths(ctx context.Context) []lang.Path {
	pathReader := &stackDecoder.PathReader{
		StateReader:  f.store,
		ModuleReader: f.moduleFeature,
		RootReader:   f.rootFeature,
	}

	return pathReader.Paths(ctx)
}

func (f *StacksFeature) Diagnostics(path string) diagnostics.Diagnostics {
	diags := diagnostics.NewDiagnostics()

	mod, err := f.store.StackRecordByPath(path)
	if err != nil {
		return diags
	}

	for source, dm := range mod.Diagnostics {
		diags.Append(source, dm.AutoloadedOnly().AsMap())
	}

	return diags
}

func (f *StacksFeature) Telemetry(path string) map[string]interface{} {
	properties := make(map[string]interface{})

	record, err := f.store.StackRecordByPath(path)
	if err != nil {
		return properties
	}

	properties["stacks"] = true

	if record.RequiredTerraformVersion != nil {
		properties["stacksTfVersion"] = record.RequiredTerraformVersion.String()
	}

	return properties
}
