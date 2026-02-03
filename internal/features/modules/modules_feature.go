// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package modules

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/algolia"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/modules/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/modules/hooks"
	"github.com/hashicorp/terraform-ls/internal/features/modules/jobs"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	"github.com/hashicorp/terraform-ls/internal/registry"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/telemetry"
	"github.com/hashicorp/terraform-schema/backend"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

// ModulesFeature groups everything related to modules. Its internal
// state keeps track of all modules in the workspace.
type ModulesFeature struct {
	Store    *state.ModuleStore
	eventbus *eventbus.EventBus
	stopFunc context.CancelFunc
	logger   *log.Logger

	rootFeature    fdecoder.RootReader
	stateStore     *globalState.StateStore
	registryClient registry.Client
	fs             jobs.ReadOnlyFS
}

func NewModulesFeature(eventbus *eventbus.EventBus, stateStore *globalState.StateStore, fs jobs.ReadOnlyFS, rootFeature fdecoder.RootReader, registryClient registry.Client) (*ModulesFeature, error) {
	store, err := state.NewModuleStore(stateStore.ProviderSchemas, stateStore.RegistryModules, stateStore.ChangeStore)
	if err != nil {
		return nil, err
	}
	discardLogger := log.New(io.Discard, "", 0)

	return &ModulesFeature{
		Store:          store,
		eventbus:       eventbus,
		stopFunc:       func() {},
		logger:         discardLogger,
		stateStore:     stateStore,
		rootFeature:    rootFeature,
		fs:             fs,
		registryClient: registryClient,
	}, nil
}

func (f *ModulesFeature) SetLogger(logger *log.Logger) {
	f.logger = logger
	f.Store.SetLogger(logger)
}

// Start starts the features separate goroutine.
// It listens to various events from the EventBus and performs corresponding actions.
func (f *ModulesFeature) Start(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	f.stopFunc = cancelFunc

	discoverDone := make(chan job.IDs, 10)
	discover := f.eventbus.OnDiscover("feature.modules", discoverDone)

	didOpenDone := make(chan job.IDs, 10)
	didOpen := f.eventbus.OnDidOpen("feature.modules", didOpenDone)

	didChangeDone := make(chan job.IDs, 10)
	didChange := f.eventbus.OnDidChange("feature.modules", didChangeDone)

	didChangeWatchedDone := make(chan job.IDs, 10)
	didChangeWatched := f.eventbus.OnDidChangeWatched("feature.modules", didChangeWatchedDone)

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

func (f *ModulesFeature) Stop() {
	f.stopFunc()
	f.logger.Print("stopped modules feature")
}

func (f *ModulesFeature) PathContext(path lang.Path) (*decoder.PathContext, error) {
	pathReader := &fdecoder.PathReader{
		StateReader: f.Store,
		RootReader:  f.rootFeature,
	}

	return pathReader.PathContext(path)
}

func (f *ModulesFeature) Paths(ctx context.Context) []lang.Path {
	pathReader := &fdecoder.PathReader{
		StateReader: f.Store,
		RootReader:  f.rootFeature,
	}

	return pathReader.Paths(ctx)
}

func (f *ModulesFeature) DeclaredModuleCalls(modPath string) (map[string]tfmod.DeclaredModuleCall, error) {
	return f.Store.DeclaredModuleCalls(modPath)
}

func (f *ModulesFeature) ProviderRequirements(modPath string) (tfmod.ProviderRequirements, error) {
	mod, err := f.Store.ModuleRecordByPath(modPath)
	if err != nil {
		return nil, err
	}

	return mod.Meta.ProviderRequirements, nil
}

func (f *ModulesFeature) CoreRequirements(modPath string) (version.Constraints, error) {
	mod, err := f.Store.ModuleRecordByPath(modPath)
	if err != nil {
		return nil, err
	}

	return mod.Meta.CoreRequirements, nil
}

func (f *ModulesFeature) ModuleInputs(modPath string) (map[string]tfmod.Variable, error) {
	mod, err := f.Store.ModuleRecordByPath(modPath)
	if err != nil {
		return nil, err
	}

	return mod.Meta.Variables, nil
}

func (f *ModulesFeature) AppendCompletionHooks(srvCtx context.Context, decoderContext decoder.DecoderContext) {
	h := hooks.Hooks{
		ModStore:       f.Store,
		RegistryClient: f.registryClient,
		Logger:         f.logger,
	}

	credentials, ok := algolia.CredentialsFromContext(srvCtx)
	if ok {
		h.AlgoliaClient = search.NewClient(credentials.AppID, credentials.APIKey)
	}

	decoderContext.CompletionHooks["CompleteLocalModuleSources"] = h.LocalModuleSources
	decoderContext.CompletionHooks["CompleteRegistryModuleSources"] = h.RegistryModuleSources
	decoderContext.CompletionHooks["CompleteRegistryModuleVersions"] = h.RegistryModuleVersions
}

func (f *ModulesFeature) Diagnostics(path string) diagnostics.Diagnostics {
	diags := diagnostics.NewDiagnostics()

	mod, err := f.Store.ModuleRecordByPath(path)
	if err != nil {
		return diags
	}

	for source, dm := range mod.ModuleDiagnostics {
		diags.Append(source, dm.AutoloadedOnly().AsMap())
	}

	return diags
}

func (f *ModulesFeature) Telemetry(path string) map[string]interface{} {
	properties := make(map[string]interface{})

	mod, err := f.Store.ModuleRecordByPath(path)
	if err != nil {
		return properties
	}

	if len(mod.Meta.CoreRequirements) > 0 {
		properties["tfRequirements"] = mod.Meta.CoreRequirements.String()
	}
	if mod.Meta.Cloud != nil {
		properties["cloud"] = true

		hostname := mod.Meta.Cloud.Hostname

		// https://developer.hashicorp.com/terraform/language/settings/terraform-cloud#usage-example
		// Required for Terraform Enterprise;
		// Defaults to app.terraform.io for HCP Terraform
		if hostname == "" {
			hostname = "app.terraform.io"
		}

		// anonymize any non-default hostnames
		if hostname != "app.terraform.io" {
			hostname = "custom-hostname"
		}

		properties["cloud.hostname"] = hostname
	}
	if mod.Meta.Backend != nil {
		properties["backend"] = mod.Meta.Backend.Type
		if data, ok := mod.Meta.Backend.Data.(*backend.Remote); ok {
			hostname := data.Hostname

			// https://developer.hashicorp.com/terraform/language/settings/backends/remote#hostname
			// Defaults to app.terraform.io for HCP Terraform
			if hostname == "" {
				hostname = "app.terraform.io"
			}

			// anonymize any non-default hostnames
			if hostname != "app.terraform.io" {
				hostname = "custom-hostname"
			}

			properties["backend.remote.hostname"] = hostname
		}
	}
	if len(mod.Meta.ProviderRequirements) > 0 {
		reqs := make(map[string]string, 0)
		for pAddr, cons := range mod.Meta.ProviderRequirements {
			if telemetry.IsPublicProvider(pAddr) {
				reqs[pAddr.String()] = cons.String()
				continue
			}

			// anonymize any unknown providers or the ones not publicly listed
			id, err := f.stateStore.ProviderSchemas.GetProviderID(pAddr)
			if err != nil {
				continue
			}
			addr := fmt.Sprintf("unlisted/%s", id)
			reqs[addr] = cons.String()
		}
		properties["providerRequirements"] = reqs
	}

	if len(mod.WriteOnlyAttributes) > 0 {
		woAttrs := make(map[string]map[string]map[string]int)

		for pAddr, stats := range mod.WriteOnlyAttributes {
			if telemetry.IsPublicProvider(pAddr) {
				woAttrs[pAddr.String()] = stats
			}
		}

		properties["writeOnlyAttributes"] = woAttrs
	}

	modId, err := f.Store.GetModuleID(mod.Path())
	if err != nil {
		return properties
	}
	properties["moduleId"] = modId

	return properties
}

// MetadataReady checks if a given module exists and if it's metadata has been
// loaded. We need the metadata to enable other features like validation for
// variables.
func (f *ModulesFeature) MetadataReady(dir document.DirHandle) (<-chan struct{}, bool, error) {
	if !f.Store.Exists(dir.Path()) {
		return nil, false, fmt.Errorf("%s: record not found", dir.Path())
	}

	return f.Store.MetadataReady(dir)
}

func (s *ModulesFeature) LocalModuleMeta(modPath string) (*tfmod.Meta, error) {
	return s.Store.LocalModuleMeta(modPath)
}
