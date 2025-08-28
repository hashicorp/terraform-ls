// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"io/fs"
	"log"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/search/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func PreloadEmbeddedSchema(ctx context.Context, logger *log.Logger, fs fs.ReadDirFS, searchStore *state.SearchStore, schemaStore *globalState.ProviderSchemaStore, searchPath string) error {
	record, err := searchStore.GetSearchRecordByPath(searchPath)

	if err != nil {
		return err
	}

	// Avoid preloading schema if it is already in progress or already known
	if record.PreloadEmbeddedSchemaState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}
	}

	err = searchStore.SetPreloadEmbeddedSchemaState(searchPath, operation.OpStateLoading)
	if err != nil {
		return err
	}
	defer searchStore.SetPreloadEmbeddedSchemaState(searchPath, operation.OpStateLoaded)

	pReqs, err := searchStore.ProviderRequirementsForModule(searchPath)
	if err != nil {
		return err
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
