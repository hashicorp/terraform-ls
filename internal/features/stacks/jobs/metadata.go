// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	earlydecoder "github.com/hashicorp/terraform-schema/earlydecoder/stacks"
)

// LoadStackMetadata loads data about the stack in a version-independent
// way that enables us to decode the rest of the configuration,
// e.g. by knowing provider versions, Terraform Core constraint etc.
func LoadStackMetadata(ctx context.Context, stackStore *state.StackStore, stackPath string) error {
	stack, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if upstream (parsing) job reported no changes

	// Avoid parsing if it is already in progress or already known
	if stack.MetaState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	err = stackStore.SetMetaState(stackPath, operation.OpStateLoading)
	if err != nil {
		return err
	}

	var mErr error
	meta, diags := earlydecoder.LoadStack(stack.Path(), stack.ParsedFiles.AsMap())
	if len(diags) > 0 {
		mErr = diags
	}

	// providerRequirements := make(map[tfaddr.Provider]version.Constraints, len(meta.ProviderRequirements))
	// for pAddr, pvc := range meta.ProviderRequirements {
	// 	// TODO: check pAddr for migrations via Registry API?
	// 	providerRequirements[pAddr] = pvc
	// }
	// meta.ProviderRequirements = providerRequirements

	// providerRefs := make(map[tfmodule.ProviderRef]tfaddr.Provider, len(meta.ProviderReferences))
	// for localRef, pAddr := range meta.ProviderReferences {
	// 	// TODO: check pAddr for migrations via Registry API?
	// 	providerRefs[localRef] = pAddr
	// }
	// meta.ProviderReferences = providerRefs

	sErr := stackStore.UpdateMetadata(stackPath, meta, mErr)
	if sErr != nil {
		return sErr
	}

	return mErr
}
