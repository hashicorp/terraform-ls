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
	var errs *multierror.Error

	pSchemaVerId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseProviderVersions(idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseProviderVersions.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, pSchemaVerId)
	}

	pSchemaId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
			return module.ObtainSchema(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type:      op.OpTypeObtainSchema.String(),
		DependsOn: job.IDs{pSchemaVerId},
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, pSchemaId)
	}

	return ids, errs.ErrorOrNil()
}
