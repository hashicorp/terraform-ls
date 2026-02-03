// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	earlydecoder "github.com/hashicorp/terraform-schema/earlydecoder/policy"
)

// LoadPolicyMetadata loads data about the policy in a version-independent
// way that enables us to decode the rest of the configuration,
// e.g. by knowing provider versions, Terraform Core constraint etc.
func LoadPolicyMetadata(ctx context.Context, policyStore *state.PolicyStore, path string) error {
	policy, err := policyStore.PolicyRecordByPath(path)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if upstream (parsing) job reported no changes

	// Avoid parsing if it is already in progress or already known
	if policy.MetaState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(path)}
	}

	err = policyStore.SetMetaState(path, op.OpStateLoading)
	if err != nil {
		return err
	}

	var mErr error
	meta, diags := earlydecoder.LoadPolicy(policy.Path(), policy.ParsedPolicyFiles.AsMap())
	if len(diags) > 0 {
		mErr = diags
	}

	sErr := policyStore.UpdateMetadata(path, meta, mErr)
	if sErr != nil {
		return sErr
	}
	return mErr
}
