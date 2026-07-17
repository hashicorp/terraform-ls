// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package policytest

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/policytest/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/jobs"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/registry"
	globalState "github.com/hashicorp/terraform-ls/internal/state"

	tfpolicytest "github.com/hashicorp/terraform-schema/policytest"
)

// PolicyTestFeature groups everything related to policytest. Its internal
// state keeps track of all policytest in the workspace.
type PolicyTestFeature struct {
	Store    *state.PolicyTestStore
	eventbus *eventbus.EventBus
	stopFunc context.CancelFunc
	logger   *log.Logger

	rootFeature fdecoder.RootReader

	stateStore     *globalState.StateStore
	registryClient registry.Client
	fs             jobs.ReadOnlyFS
}

func NewPolicyTestFeature(eventbus *eventbus.EventBus, stateStore *globalState.StateStore, fs jobs.ReadOnlyFS, rootFeature fdecoder.RootReader) (*PolicyTestFeature, error) {
	store, err := state.NewPolicyTestStore(stateStore.ChangeStore)
	if err != nil {
		return nil, err
	}
	discardLogger := log.New(io.Discard, "", 0)

	return &PolicyTestFeature{
		Store:       store,
		eventbus:    eventbus,
		stopFunc:    func() {},
		logger:      discardLogger,
		stateStore:  stateStore,
		rootFeature: rootFeature,
		fs:          fs,
	}, nil
}

func (f *PolicyTestFeature) SetLogger(logger *log.Logger) {
	f.logger = logger
	f.Store.SetLogger(logger)
}

// Start starts the features separate goroutine.
// It listens to various events from the EventBus and performs corresponding actions.
func (f *PolicyTestFeature) Start(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	f.stopFunc = cancelFunc

	discoverDone := make(chan job.IDs, 10)
	discover := f.eventbus.OnDiscover("feature.policytest", discoverDone)

	didOpenDone := make(chan job.IDs, 10)
	didOpen := f.eventbus.OnDidOpen("feature.policytest", didOpenDone)

	didChangeDone := make(chan job.IDs, 10)
	didChange := f.eventbus.OnDidChange("feature.policytest", didChangeDone)

	didChangeWatchedDone := make(chan job.IDs, 10)
	didChangeWatched := f.eventbus.OnDidChangeWatched("feature.policytest", didChangeWatchedDone)

	go func() {
		for {
			select {
			case discover := <-discover:
				// TODO? collect errors
				f.discover(discover.Path, discover.Files)
				discoverDone <- job.IDs{}
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

func (f *PolicyTestFeature) Stop() {
	f.stopFunc()
	f.logger.Print("stopped policytest feature")
}

func (f *PolicyTestFeature) PathContext(path lang.Path) (*decoder.PathContext, error) {
	pathReader := &fdecoder.PathReader{
		StateReader: f.Store,
		RootReader:  f.rootFeature,
	}

	return pathReader.PathContext(path)
}

func (f *PolicyTestFeature) Paths(ctx context.Context) []lang.Path {
	pathReader := &fdecoder.PathReader{
		StateReader: f.Store,
		RootReader:  f.rootFeature,
	}

	return pathReader.Paths(ctx)
}

func (f *PolicyTestFeature) Diagnostics(path string) diagnostics.Diagnostics {
	diags := diagnostics.NewDiagnostics()

	policytest, err := f.Store.PolicyTestRecordByPath(path)
	if err != nil {
		return diags
	}

	for source, dm := range policytest.PolicyTestDiagnostics {
		diags.Append(source, dm.AutoloadedOnly().AsMap())
	}

	return diags
}

func (f *PolicyTestFeature) Telemetry(path string) map[string]interface{} {
	properties := make(map[string]interface{})

	policytest, err := f.Store.PolicyTestRecordByPath(path)
	if err != nil {
		return properties
	}

	policytestId, err := f.Store.GetPolicyTestID(policytest.Path())
	if err != nil {
		return properties
	}
	properties["policytestId"] = policytestId

	return properties
}

// MetadataReady checks if a given policytest exists and if it's metadata has been
// loaded. We need the metadata to enable other features like validation for
// variables.
func (f *PolicyTestFeature) MetadataReady(dir document.DirHandle) (<-chan struct{}, bool, error) {
	if !f.Store.Exists(dir.Path()) {
		return nil, false, fmt.Errorf("%s: record not found", dir.Path())
	}

	return f.Store.MetadataReady(dir)
}

func (s *PolicyTestFeature) LocalPolicyTestMeta(policytestPath string) (*tfpolicytest.Meta, error) {
	return s.Store.LocalPolicyTestMeta(policytestPath)
}
