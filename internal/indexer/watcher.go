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

	modManifestId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleManifest(idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseModuleManifest.String(),
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			return idx.decodeInstalledModuleCalls(modHandle)
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, modManifestId)

	return ids, nil
}

func (idx *Indexer) PluginLockChanged(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseProviderVersions(idx.fs, idx.modStore, modHandle.Path())
		},
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			ids := make(job.IDs, 0)

			mod, err := idx.modStore.ModuleByPath(modHandle.Path())
			if err != nil {
				return ids, err
			}

			exist, err := idx.schemaStore.AllSchemasExist(mod.Meta.ProviderRequirements)
			if err != nil {
				return ids, err
			}
			if exist {
				// avoid obtaining schemas if we already have it
				return ids, nil
			}

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

			return ids, nil
		},
		Type: op.OpTypeParseProviderVersions.String(),
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
