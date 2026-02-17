// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	earlydecoder "github.com/hashicorp/terraform-schema/earlydecoder/policytest"
)

// LoadPolicyTestMetadata loads data about the policytest in a version-independent
// way that enables us to decode the rest of the configuration,
// e.g. by knowing provider versions, Terraform Core constraint etc.
func LoadPolicyTestMetadata(ctx context.Context, policytestStore *state.PolicyTestStore, path string) error {
	policytest, err := policytestStore.PolicyTestRecordByPath(path)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if upstream (parsing) job reported no changes

	// Avoid parsing if it is already in progress or already known
	if policytest.MetaState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(path)}
	}

	err = policytestStore.SetMetaState(path, op.OpStateLoading)
	if err != nil {
		return err
	}

	var mErr error
	meta, diags := earlydecoder.LoadPolicyTest(policytest.Path(), policytest.ParsedPolicyTestFiles.AsMap())
	if len(diags) > 0 {
		mErr = diags
	}

	sErr := policytestStore.UpdateMetadata(path, meta, mErr)
	if sErr != nil {
		return sErr
	}
	return mErr
}
