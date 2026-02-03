// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

// GetTerraformVersion obtains "installed" Terraform version
// which can inform what version of core schema to pick.
// Knowing the version is not required though as we can rely on
// the constraint in `required_version` (as parsed via
// [LoadModuleMetadata] and compare it against known released versions.
func GetTerraformVersion(ctx context.Context, rootStore *state.RootStore, modPath string) error {
	mod, err := rootStore.RootRecordByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid getting version if getting is already in progress or already known
	if mod.TerraformVersionState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = rootStore.SetTerraformVersionState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}
	defer rootStore.SetTerraformVersionState(modPath, op.OpStateLoaded)

	tfExec, err := module.TerraformExecutorForModule(ctx, mod.Path())
	if err != nil {
		sErr := rootStore.UpdateTerraformAndProviderVersions(modPath, nil, nil, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	v, pv, err := tfExec.Version(ctx)

	// TODO: Remove and rely purely on ParseProviderVersions
	// In most cases we get the provider version from the datadir/lockfile
	// but there is an edge case with custom plugin location
	// when this may not be available, so leveraging versions
	// from "terraform version" accounts for this.
	// See https://github.com/hashicorp/terraform-ls/issues/24
	pVersions := providerVersionsFromTfVersion(pv)

	sErr := rootStore.UpdateTerraformAndProviderVersions(modPath, v, pVersions, err)
	if sErr != nil {
		return sErr
	}

	return err
}

func providerVersionsFromTfVersion(pv map[string]*version.Version) map[tfaddr.Provider]*version.Version {
	m := make(map[tfaddr.Provider]*version.Version, 0)

	for rawAddr, v := range pv {
		pAddr, err := tfaddr.ParseProviderSource(rawAddr)
		if err != nil {
			// skip unparsable address
			continue
		}
		if pAddr.IsLegacy() {
			// TODO: check for migrations via Registry API?
		}
		m[pAddr] = v
	}

	return m
}
