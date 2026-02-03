// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"io/fs"
	"log"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

func PreloadEmbeddedSchema(ctx context.Context, logger *log.Logger, fs fs.ReadDirFS, stackStore *state.StackStore, schemaStore *globalState.ProviderSchemaStore, stackPath string) error {
	record, err := stackStore.StackRecordByPath(stackPath)

	if err != nil {
		return err
	}

	// Avoid preloading schema if it is already in progress or already known
	if record.PreloadEmbeddedSchemaState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	err = stackStore.SetPreloadEmbeddedSchemaState(stackPath, operation.OpStateLoading)
	if err != nil {
		return err
	}
	defer stackStore.SetPreloadEmbeddedSchemaState(stackPath, operation.OpStateLoaded)

	pReqs := make(map[tfaddr.Provider]version.Constraints, len(record.Meta.ProviderRequirements))
	for _, req := range record.Meta.ProviderRequirements {
		pReqs[req.Source] = req.VersionConstraints
	}

	missingReqs, err := schemaStore.MissingSchemas(pReqs)
	if err != nil {
		return err
	}

	if len(missingReqs) == 0 {
		// avoid preloading any schemas if we already have all
		return nil
	}

	for _, pAddr := range missingReqs {
		err := globalState.PreloadSchemaForProviderAddr(ctx, pAddr, fs, schemaStore, logger)
		if err != nil {
			return err
		}
	}

	return nil

}
