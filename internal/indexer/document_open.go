// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package indexer

import (
	"context"

	"github.com/hashicorp/go-multierror"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (idx *Indexer) DocumentOpened(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	mod, err := idx.modStore.ModuleByPath(modHandle.Path())
	if err != nil {
		return nil, err
	}

	ids := make(job.IDs, 0)
	var errs *multierror.Error

	if mod.TerraformVersionState == op.OpStateUnknown {
		_, err := idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
				return module.GetTerraformVersion(ctx, idx.modStore, modHandle.Path())
			},
			Type: op.OpTypeGetTerraformVersion.String(),
		})
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		// Given that getting version may take time and we only use it to
		// enhance the UX, we ignore the outcome (job ID) here
		// to avoid delays when documents of new modules are open.
	}

	parseId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleConfiguration(ctx, idx.fs, idx.modStore, modHandle.Path())
		},
		Type:        op.OpTypeParseModuleConfiguration.String(),
		IgnoreState: true,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseId)

	modIds, err := idx.decodeModule(ctx, modHandle, job.IDs{parseId}, true)
	if err != nil {
		return ids, err
	}
	ids = append(ids, modIds...)

	parseVarsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseVariables(ctx, idx.fs, idx.modStore, modHandle.Path())
		},
		Type:        op.OpTypeParseVariables.String(),
		IgnoreState: true,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseVarsId)

	validationOptions, err := lsctx.ValidationOptions(ctx)
	if err != nil {
		return ids, err
	}

	if validationOptions.EnableEnhancedValidation {
		_, err = idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.SchemaVariablesValidation(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
			},
			Type:        op.OpTypeSchemaVarsValidation.String(),
			DependsOn:   append(modIds, parseVarsId),
			IgnoreState: true,
		})
		if err != nil {
			return ids, err
		}
	}

	parseTestId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseTests(ctx, idx.fs, idx.modStore, modHandle.Path())
		},
		Type:        op.OpTypeParseTests.String(),
		IgnoreState: true,
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, parseTestId)
	}

	varsRefsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeVarsReferences(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type:      op.OpTypeDecodeVarsReferences.String(),
		DependsOn: job.IDs{parseVarsId},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, varsRefsId)

	return ids, errs.ErrorOrNil()
}
