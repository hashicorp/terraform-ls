package indexer

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (idx *Indexer) DocumentOpened(modHandle document.DirHandle) (job.IDs, error) {
	mod, err := idx.modStore.ModuleByPath(modHandle.Path())
	if err != nil {
		return nil, err
	}

	ids := make(job.IDs, 0)
	var errs *multierror.Error

	if mod.TerraformVersionState == op.OpStateUnknown {
		_, err := idx.jobStore.EnqueueJob(job.Job{
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

	parseId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleConfiguration(ctx, idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseModuleConfiguration.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseId)

	modIds, err := idx.decodeModule(modHandle, job.IDs{parseId})
	if err != nil {
		return ids, err
	}
	ids = append(ids, modIds...)

	parseVarsId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseVariables(ctx, idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseVariables.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseVarsId)

	varsRefsId, err := idx.jobStore.EnqueueJob(job.Job{
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
