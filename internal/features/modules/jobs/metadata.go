// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/earlydecoder"
	tfmodule "github.com/hashicorp/terraform-schema/module"
)

// LoadModuleMetadata loads data about the module in a version-independent
// way that enables us to decode the rest of the configuration,
// e.g. by knowing provider versions, Terraform Core constraint etc.
func LoadModuleMetadata(ctx context.Context, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if upstream (parsing) job reported no changes

	// Avoid parsing if it is already in progress or already known
	if mod.MetaState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetMetaState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	var mErr error
	meta, diags := earlydecoder.LoadModule(mod.Path(), mod.ParsedModuleFiles.AsMap())
	if len(diags) > 0 {
		mErr = diags
	}

	providerRequirements := make(map[tfaddr.Provider]version.Constraints, len(meta.ProviderRequirements))
	for pAddr, pvc := range meta.ProviderRequirements {
		// TODO: check pAddr for migrations via Registry API?
		providerRequirements[pAddr] = pvc
	}
	meta.ProviderRequirements = providerRequirements

	providerRefs := make(map[tfmodule.ProviderRef]tfaddr.Provider, len(meta.ProviderReferences))
	for localRef, pAddr := range meta.ProviderReferences {
		// TODO: check pAddr for migrations via Registry API?
		providerRefs[localRef] = pAddr
	}
	meta.ProviderReferences = providerRefs

	sErr := modStore.UpdateMetadata(modPath, meta, mErr)
	if sErr != nil {
		return sErr
	}
	return mErr
}
