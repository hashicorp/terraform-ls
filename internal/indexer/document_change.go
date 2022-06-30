package indexer

import (
	"context"

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
		Defer: func(ctx context.Context, jobErr error) job.IDs {
			ids, err := idx.decodeModule(ctx, modHandle)
			if err != nil {
				idx.logger.Printf("error: %s", err)
			}
			return ids
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
		Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
			id, err := idx.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeVarsReferences(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
				},
				Type: op.OpTypeDecodeVarsReferences.String(),
			})
			if err != nil {
				return
			}
			ids = append(ids, id)
			return
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
		Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
			id, err := idx.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeReferenceTargets(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
				},
				Type: op.OpTypeDecodeReferenceTargets.String(),
			})
			if err != nil {
				return
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
				return
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
				return
			}
			ids = append(ids, id)

			return
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	return ids, nil
}
