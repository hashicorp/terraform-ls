// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package variables

import (
	"context"
	"io"
	"log"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/variables/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/variables/jobs"
	"github.com/hashicorp/terraform-ls/internal/features/variables/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
)

// VariablesFeature groups everything related to variables. Its internal
// state keeps track of all variable definition files in the workspace.
type VariablesFeature struct {
	store    *state.VariableStore
	eventbus *eventbus.EventBus
	stopFunc context.CancelFunc
	logger   *log.Logger

	moduleFeature fdecoder.ModuleReader
	stateStore    *globalState.StateStore
	fs            jobs.ReadOnlyFS
}

func NewVariablesFeature(eventbus *eventbus.EventBus, stateStore *globalState.StateStore, fs jobs.ReadOnlyFS, moduleFeature fdecoder.ModuleReader) (*VariablesFeature, error) {
	store, err := state.NewVariableStore(stateStore.ChangeStore)
	if err != nil {
		return nil, err
	}
	discardLogger := log.New(io.Discard, "", 0)

	return &VariablesFeature{
		store:         store,
		eventbus:      eventbus,
		stopFunc:      func() {},
		logger:        discardLogger,
		moduleFeature: moduleFeature,
		stateStore:    stateStore,
		fs:            fs,
	}, nil
}

func (f *VariablesFeature) SetLogger(logger *log.Logger) {
	f.logger = logger
	f.store.SetLogger(logger)
}

// Start starts the features separate goroutine.
// It listens to various events from the EventBus and performs corresponding actions.
func (f *VariablesFeature) Start(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	f.stopFunc = cancelFunc

	discover := f.eventbus.OnDiscover("feature.variables", nil)

	didOpenDone := make(chan job.IDs, 10)
	didOpen := f.eventbus.OnDidOpen("feature.variables", didOpenDone)

	didChangeDone := make(chan job.IDs, 10)
	didChange := f.eventbus.OnDidChange("feature.variables", didChangeDone)

	didChangeWatchedDone := make(chan job.IDs, 10)
	didChangeWatched := f.eventbus.OnDidChangeWatched("feature.variables", didChangeWatchedDone)

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

func (f *VariablesFeature) Stop() {
	f.stopFunc()
	f.logger.Print("stopped variables feature")
}

func (f *VariablesFeature) PathContext(path lang.Path) (*decoder.PathContext, error) {
	pathReader := &fdecoder.PathReader{
		StateReader:  f.store,
		ModuleReader: f.moduleFeature,
	}

	return pathReader.PathContext(path)
}

func (f *VariablesFeature) Paths(ctx context.Context) []lang.Path {
	pathReader := &fdecoder.PathReader{
		StateReader:  f.store,
		ModuleReader: f.moduleFeature,
	}

	return pathReader.Paths(ctx)
}

func (f *VariablesFeature) Diagnostics(path string) diagnostics.Diagnostics {
	diags := diagnostics.NewDiagnostics()

	mod, err := f.store.VariableRecordByPath(path)
	if err != nil {
		return diags
	}

	for source, dm := range mod.VarsDiagnostics {
		diags.Append(source, dm.AutoloadedOnly().AsMap())
	}

	return diags
}
