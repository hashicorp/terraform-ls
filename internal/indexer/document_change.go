package indexer

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (idx *Indexer) DocumentChanged(modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleConfiguration(idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseModuleConfiguration.String(),
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			return idx.decodeModule(ctx, modHandle)
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	id, err = idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseVariables(idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseVariables.String(),
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			ids := make(job.IDs, 0)
			id, err := idx.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeVarsReferences(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
				},
				Type: op.OpTypeDecodeVarsReferences.String(),
			})
			if err != nil {
				return ids, err
			}
			ids = append(ids, id)
			return ids, nil
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	return ids, nil
}

func (idx *Indexer) decodeModule(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.LoadModuleMetadata(idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeLoadModuleMetadata.String(),
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			ids := make(job.IDs, 0)

			var errs *multierror.Error

			mcIds, err := idx.decodeDeclaredModuleCalls(modHandle)(ctx, jobErr)
			if err != nil {
				errs = multierror.Append(errs, err)
			} else {
				ids = append(ids, mcIds...)
			}

			id, err := idx.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeReferenceTargets(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
				},
				Type: op.OpTypeDecodeReferenceTargets.String(),
			})
			if err != nil {
				return ids, err
			}
			ids = append(ids, id)

			id, err = idx.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeReferenceOrigins(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
				},
				Type: op.OpTypeDecodeReferenceOrigins.String(),
			})
			if err != nil {
				return ids, err
			}
			ids = append(ids, id)

			id, err = idx.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.GetModuleDataFromRegistry(ctx, idx.registryClient,
						idx.modStore, idx.registryModStore, modHandle.Path())
				},
				Priority: job.LowPriority,
				Type:     op.OpTypeGetModuleDataFromRegistry.String(),
			})
			if err != nil {
				return ids, err
			}

			ids = append(ids, id)
			return ids, nil
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	return ids, nil
}
