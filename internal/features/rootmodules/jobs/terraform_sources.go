// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// ParseTerraformSources parses the NEW* "module manifest" which
// contains records of installed modules, e.g. where they're
// installed on the filesystem.
// This is useful for processing any modules which are not local
// nor hosted in the Registry (which would be handled by
// [GetModuleDataFromRegistry]).
// NEW* as there is a new terraform-sources.json file format which currently only exists for stacks.
func ParseTerraformSources(ctx context.Context, fs ReadOnlyFS, rootStore *state.RootStore, modPath string) error {
	mod, err := rootStore.RootRecordByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid parsing if it is already in progress or already known
	if mod.TerraformSourcesState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = rootStore.SetTerraformSourcesState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	tfs, err := datadir.ParseTerraformSourcesFromFile(modPath)
	if err != nil {
		err := fmt.Errorf("failed to parse terraform sources: %w", err)
		sErr := rootStore.UpdateTerraformSources(modPath, nil, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	sErr := rootStore.UpdateTerraformSources(modPath, tfs, err)

	if sErr != nil {
		return sErr
	}
	return err
}
