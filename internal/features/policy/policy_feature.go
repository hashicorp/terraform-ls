// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/policy/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/policy/jobs"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/registry"
	globalState "github.com/hashicorp/terraform-ls/internal/state"

	tfpolicy "github.com/hashicorp/terraform-schema/policy"
)

// PolicyFeature groups everything related to policy. Its internal
// state keeps track of all policy in the workspace.
type PolicyFeature struct {
	Store    *state.PolicyStore
	eventbus *eventbus.EventBus
	stopFunc context.CancelFunc
	logger   *log.Logger

	rootFeature fdecoder.RootReader

	stateStore     *globalState.StateStore
	registryClient registry.Client
	fs             jobs.ReadOnlyFS
}

func NewPolicyFeature(eventbus *eventbus.EventBus, stateStore *globalState.StateStore, fs jobs.ReadOnlyFS, rootFeature fdecoder.RootReader) (*PolicyFeature, error) {
	store, err := state.NewPolicyStore(stateStore.ChangeStore)
	if err != nil {
		return nil, err
	}
	discardLogger := log.New(io.Discard, "", 0)

	return &PolicyFeature{
		Store:       store,
		eventbus:    eventbus,
		stopFunc:    func() {},
		logger:      discardLogger,
		stateStore:  stateStore,
		rootFeature: rootFeature,
		fs:          fs,
	}, nil
}

func (f *PolicyFeature) SetLogger(logger *log.Logger) {
	f.logger = logger
	f.Store.SetLogger(logger)
}

// Start starts the features separate goroutine.
// It listens to various events from the EventBus and performs corresponding actions.
func (f *PolicyFeature) Start(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	f.stopFunc = cancelFunc

	discoverDone := make(chan job.IDs, 10)
	discover := f.eventbus.OnDiscover("feature.policy", discoverDone)

	didOpenDone := make(chan job.IDs, 10)
	didOpen := f.eventbus.OnDidOpen("feature.policy", didOpenDone)

	didChangeDone := make(chan job.IDs, 10)
	didChange := f.eventbus.OnDidChange("feature.policy", didChangeDone)

	didChangeWatchedDone := make(chan job.IDs, 10)
	didChangeWatched := f.eventbus.OnDidChangeWatched("feature.policy", didChangeWatchedDone)

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

func (f *PolicyFeature) Stop() {
	f.stopFunc()
	f.logger.Print("stopped policy feature")
}

func (f *PolicyFeature) PathContext(path lang.Path) (*decoder.PathContext, error) {
	pathReader := &fdecoder.PathReader{
		StateReader: f.Store,
		RootReader:  f.rootFeature,
	}

	return pathReader.PathContext(path)
}

func (f *PolicyFeature) Paths(ctx context.Context) []lang.Path {
	pathReader := &fdecoder.PathReader{
		StateReader: f.Store,
		RootReader:  f.rootFeature,
	}

	return pathReader.Paths(ctx)
}

func (f *PolicyFeature) CoreRequirements(policyPath string) (version.Constraints, error) {
	policy, err := f.Store.PolicyRecordByPath(policyPath)
	if err != nil {
		return nil, err
	}

	return policy.Meta.CoreRequirements, nil
}

func (f *PolicyFeature) Diagnostics(path string) diagnostics.Diagnostics {
	diags := diagnostics.NewDiagnostics()

	policy, err := f.Store.PolicyRecordByPath(path)
	if err != nil {
		return diags
	}

	for source, dm := range policy.PolicyDiagnostics {
		diags.Append(source, dm.AutoloadedOnly().AsMap())
	}

	return diags
}

func (f *PolicyFeature) Telemetry(path string) map[string]interface{} {
	properties := make(map[string]interface{})

	policy, err := f.Store.PolicyRecordByPath(path)
	if err != nil {
		return properties
	}

	if len(policy.Meta.CoreRequirements) > 0 {
		properties["tfRequirements"] = policy.Meta.CoreRequirements.String()
	}

	policyId, err := f.Store.GetPolicyID(policy.Path())
	if err != nil {
		return properties
	}
	properties["policyId"] = policyId

	return properties
}

// MetadataReady checks if a given policy exists and if it's metadata has been
// loaded. We need the metadata to enable other features like validation for
// variables.
func (f *PolicyFeature) MetadataReady(dir document.DirHandle) (<-chan struct{}, bool, error) {
	if !f.Store.Exists(dir.Path()) {
		return nil, false, fmt.Errorf("%s: record not found", dir.Path())
	}

	return f.Store.MetadataReady(dir)
}

func (s *PolicyFeature) LocalPolicyMeta(policyPath string) (*tfpolicy.Meta, error) {
	return s.Store.LocalPolicyMeta(policyPath)
}
