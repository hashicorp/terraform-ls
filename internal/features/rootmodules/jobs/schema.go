// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

// ObtainSchema obtains provider schemas via Terraform CLI.
// This is useful if we do not have the schemas available
// from the embedded FS (i.e. in [PreloadEmbeddedSchema]).
func ObtainSchema(ctx context.Context, rootStore *state.RootStore, schemaStore *globalState.ProviderSchemaStore, modPath string) error {
	record, err := rootStore.RootRecordByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid obtaining schema if it is already in progress or already known
	if record.ProviderSchemaState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	// We rely on the state to see if the job already ran
	// 1. it will run whenever we open a root module for the first time
	// 2. it will run when we detect changes to a lockfile

	tfExec, err := module.TerraformExecutorForModule(ctx, modPath)
	if err != nil {
		sErr := rootStore.FinishProviderSchemaLoading(modPath, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	ps, err := tfExec.ProviderSchemas(ctx)
	if err != nil {
		sErr := rootStore.FinishProviderSchemaLoading(modPath, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	for rawAddr, pJsonSchema := range ps.Schemas {
		pAddr, err := tfaddr.ParseProviderSource(rawAddr)
		if err != nil {
			// skip unparsable address
			continue
		}

		if pAddr.IsLegacy() {
			// TODO: check for migrations via Registry API?
		}

		pSchema := tfschema.ProviderSchemaFromJson(pJsonSchema, pAddr)

		err = schemaStore.AddLocalSchema(modPath, pAddr, pSchema)
		if err != nil {
			return err
		}
	}

	err = rootStore.FinishProviderSchemaLoading(modPath, nil)
	if err != nil {
		return err
	}

	return nil
}
