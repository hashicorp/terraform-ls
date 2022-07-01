package indexer

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (idx *Indexer) ModuleManifestChanged(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleManifest(idx.fs, idx.modStore, modHandle.Path())
		},
		Type:  op.OpTypeParseModuleManifest.String(),
		Defer: decodeInstalledModuleCalls(idx.fs, idx.modStore, idx.schemaStore, modHandle.Path()),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	return ids, nil
}

func (idx *Indexer) PluginLockChanged(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
			eo, ok := exec.ExecutorOptsFromContext(ctx)
			if ok {
				ctx = exec.WithExecutorOpts(ctx, eo)
			}

			return module.ObtainSchema(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type: op.OpTypeObtainSchema.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	id, err = idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
			eo, ok := exec.ExecutorOptsFromContext(ctx)
			if ok {
				ctx = exec.WithExecutorOpts(ctx, eo)
			}

			return module.GetTerraformVersion(ctx, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeGetTerraformVersion.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	return ids, nil
}
