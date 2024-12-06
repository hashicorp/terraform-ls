// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/tests/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	earlydecoder "github.com/hashicorp/terraform-schema/earlydecoder/tests"
	"github.com/hashicorp/terraform-schema/test"
)

// LoadTestMetadata loads data about the test in a version-independent
// way that enables us to decode the rest of the configuration,
// e.g. by knowing provider versions, etc.
func LoadTestMetadata(ctx context.Context, testStore *state.TestStore, testPath string) error {
	record, err := testStore.TestRecordByPath(testPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if upstream (parsing) job reported no changes

	// Avoid parsing if it is already in progress or already known
	if record.MetaState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(testPath)}
	}

	err = testStore.SetMetaState(testPath, operation.OpStateLoading)
	if err != nil {
		return err
	}

	var diags hcl.Diagnostics
	meta := map[string]*test.Meta{}
	for filename, file := range record.ParsedFiles.AsMap() {
		fMeta, fDiags := earlydecoder.LoadTest(record.Path(), filename, file)
		diags = append(diags, fDiags...)
		meta[filename] = fMeta
	}

	var mErr error
	if len(diags) > 0 {
		mErr = diags
	}

	sErr := testStore.UpdateMetadata(testPath, meta, mErr)
	if sErr != nil {
		return sErr
	}

	return mErr
}
