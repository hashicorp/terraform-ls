// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package rootmodules

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/jobs"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/telemetry"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

// RootModulesFeature groups everything related to root modules. Its internal
// state keeps track of all root modules in the workspace. A root module is
// usually the directory where you would run `terraform init` and where the
// `.terraform` directory and `.terraform.lock.hcl` are located.
//
// The feature listens to events from the EventBus to update its state and
// act on lockfile changes. It also provides methods to query root modules
// for the installed providers, modules, and Terraform version.
type RootModulesFeature struct {
	Store    *state.RootStore
	eventbus *eventbus.EventBus
	stopFunc context.CancelFunc
	logger   *log.Logger

	tfExecFactory exec.ExecutorFactory
	stateStore    *globalState.StateStore
	fs            jobs.ReadOnlyFS
}

func NewRootModulesFeature(eventbus *eventbus.EventBus, stateStore *globalState.StateStore, fs jobs.ReadOnlyFS, tfExecFactory exec.ExecutorFactory) (*RootModulesFeature, error) {
	store, err := state.NewRootStore(stateStore.ChangeStore, stateStore.ProviderSchemas)
	if err != nil {
		return nil, err
	}
	discardLogger := log.New(io.Discard, "", 0)

	return &RootModulesFeature{
		Store:         store,
		eventbus:      eventbus,
		stopFunc:      func() {},
		logger:        discardLogger,
		tfExecFactory: tfExecFactory,
		stateStore:    stateStore,
		fs:            fs,
	}, nil
}

func (f *RootModulesFeature) SetLogger(logger *log.Logger) {
	f.logger = logger
	f.Store.SetLogger(logger)
}

// Start starts the features separate goroutine.
// It listens to various events from the EventBus and performs corresponding actions.
func (f *RootModulesFeature) Start(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	f.stopFunc = cancelFunc

	discoverDone := make(chan job.IDs, 10)
	discover := f.eventbus.OnDiscover("feature.rootmodules", discoverDone)

	didOpenDone := make(chan job.IDs, 10)
	didOpen := f.eventbus.OnDidOpen("feature.rootmodules", didOpenDone)

	manifestChangeDone := make(chan job.IDs, 10)
	manifestChange := f.eventbus.OnManifestChange("feature.rootmodules", manifestChangeDone)

	pluginLockChangeDone := make(chan job.IDs, 10)
	pluginLockChange := f.eventbus.OnPluginLockChange("feature.rootmodules", pluginLockChangeDone)

	go func() {
		for {
			select {
			case discover := <-discover:
				// TODO? collect errors
				f.discover(discover.Path, discover.Files)
				discoverDone <- job.IDs{}
			case didOpen := <-didOpen:
				// TODO? collect errors
				spawnedIds, _ := f.didOpen(didOpen.Context, didOpen.Dir)
				didOpenDone <- spawnedIds
			case manifestChange := <-manifestChange:
				// TODO? collect errors
				spawnedIds, _ := f.manifestChange(manifestChange.Context, manifestChange.Dir, manifestChange.ChangeType)
				manifestChangeDone <- spawnedIds
			case pluginLockChange := <-pluginLockChange:
				// TODO? collect errors
				spawnedIds, _ := f.pluginLockChange(pluginLockChange.Context, pluginLockChange.Dir)
				pluginLockChangeDone <- spawnedIds

			case <-ctx.Done():
				return
			}
		}
	}()
}

func (f *RootModulesFeature) Stop() {
	f.stopFunc()
	f.logger.Print("stopped root modules feature")
}

// InstalledModuleCalls returns the installed module based on the module manifest
func (f *RootModulesFeature) InstalledModuleCalls(modPath string) (map[string]tfmod.InstalledModuleCall, error) {
	return f.Store.InstalledModuleCalls(modPath)
}

// TerraformVersion tries to find a modules Terraform version on a best effort basis.
// If a root module exists at the given path, it will return the Terraform
// version of that root module. If not, it will return the version of any
// of the other root modules.
func (f *RootModulesFeature) TerraformVersion(modPath string) *version.Version {
	record, err := f.Store.RootRecordByPath(modPath)
	if err != nil {
		if globalState.IsRecordNotFound(err) {
			// TODO try a proximity search to find the closest root module
			record, err = f.Store.RecordWithVersion()
			if err != nil {
				return nil
			}

			return record.TerraformVersion
		}

		return nil
	}

	return record.TerraformVersion
}

// InstalledProviders returns the installed providers for the given module path
func (f *RootModulesFeature) InstalledProviders(modPath string) (map[tfaddr.Provider]*version.Version, error) {
	record, err := f.Store.RootRecordByPath(modPath)
	if err != nil {
		return nil, err
	}

	return record.InstalledProviders, nil
}

func (f *RootModulesFeature) CallersOfModule(modPath string) ([]string, error) {
	return f.Store.CallersOfModule(modPath)
}

func (f *RootModulesFeature) Telemetry(path string) map[string]interface{} {
	properties := make(map[string]interface{})

	record, err := f.Store.RootRecordByPath(path)
	if err != nil {
		return properties
	}

	if record.TerraformVersion != nil {
		properties["tfVersion"] = record.TerraformVersion.String()
	}
	if len(record.InstalledProviders) > 0 {
		installedProviders := make(map[string]string, 0)
		for pAddr, pv := range record.InstalledProviders {
			if telemetry.IsPublicProvider(pAddr) {
				versionString := ""
				if pv != nil {
					versionString = pv.String()
				}
				installedProviders[pAddr.String()] = versionString
				continue
			}

			// anonymize any unknown providers or the ones not publicly listed
			id, err := f.stateStore.ProviderSchemas.GetProviderID(pAddr)
			if err != nil {
				continue
			}
			addr := fmt.Sprintf("unlisted/%s", id)
			installedProviders[addr] = ""
		}
		properties["installedProviders"] = installedProviders
	}

	return properties
}

// InstalledModulePath checks the installed modules in the given root module
// for the given normalized source address.
//
// If the module is installed, it returns the path to the module installation
// directory on disk.
func (f *RootModulesFeature) InstalledModulePath(rootPath string, normalizedSource string) (string, bool) {
	record, err := f.Store.RootRecordByPath(rootPath)
	if err != nil {
		return "", false
	}

	dir, ok := record.InstalledModules[normalizedSource]
	if !ok {
		return "", false
	}

	return dir, true
}
