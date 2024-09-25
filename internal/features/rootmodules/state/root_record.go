// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// RootRecord contains all information about a module root path, like
// anything related to .terraform/ or .terraform.lock.hcl.
type RootRecord struct {
	path string

	// ProviderSchemaState tracks if we tried loading all provider schemas
	// that this module is using via Terraform CLI
	ProviderSchemaState op.OpState
	ProviderSchemaErr   error

	ModManifest      *datadir.ModuleManifest
	ModManifestErr   error
	ModManifestState op.OpState

	TerraformSources      *datadir.TerraformSources
	TerraformSourcesErr   error
	TerraformSourcesState op.OpState

	// InstalledModules is a map of normalized source addresses from the
	// manifest to the path of the local directory where the module is installed
	InstalledModules InstalledModules

	TerraformVersion      *version.Version
	TerraformVersionErr   error
	TerraformVersionState op.OpState

	InstalledProviders      InstalledProviders
	InstalledProvidersErr   error
	InstalledProvidersState op.OpState
}

func (m *RootRecord) Copy() *RootRecord {
	if m == nil {
		return nil
	}
	newRecord := &RootRecord{
		path: m.path,

		ProviderSchemaErr:   m.ProviderSchemaErr,
		ProviderSchemaState: m.ProviderSchemaState,

		ModManifest:      m.ModManifest.Copy(),
		ModManifestErr:   m.ModManifestErr,
		ModManifestState: m.ModManifestState,

		TerraformSources:      m.TerraformSources.Copy(),
		TerraformSourcesErr:   m.TerraformSourcesErr,
		TerraformSourcesState: m.TerraformSourcesState,

		// version.Version is practically immutable once parsed
		TerraformVersion:      m.TerraformVersion,
		TerraformVersionErr:   m.TerraformVersionErr,
		TerraformVersionState: m.TerraformVersionState,

		InstalledProvidersErr:   m.InstalledProvidersErr,
		InstalledProvidersState: m.InstalledProvidersState,
	}

	if m.InstalledProviders != nil {
		newRecord.InstalledProviders = make(InstalledProviders, len(m.InstalledProviders))
		for addr, pv := range m.InstalledProviders {
			// version.Version is practically immutable once parsed
			newRecord.InstalledProviders[addr] = pv
		}
	}

	if m.InstalledModules != nil {
		newRecord.InstalledModules = make(InstalledModules, len(m.InstalledModules))
		for source, dir := range m.InstalledModules {
			newRecord.InstalledModules[source] = dir
		}
	}

	return newRecord
}

func (m *RootRecord) Path() string {
	return m.path
}

func newRootRecord(path string) *RootRecord {
	return &RootRecord{
		path:                    path,
		ProviderSchemaState:     op.OpStateUnknown,
		ModManifestState:        op.OpStateUnknown,
		TerraformSourcesState:   op.OpStateUnknown,
		TerraformVersionState:   op.OpStateUnknown,
		InstalledProvidersState: op.OpStateUnknown,
	}
}

// NewRootRecordTest is a test helper to create a new Module object
func NewRootRecordTest(path string) *RootRecord {
	return &RootRecord{
		path: path,
	}
}
