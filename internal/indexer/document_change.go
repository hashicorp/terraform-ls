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

	return ids, nil
}

func (idx *Indexer) decodeModule(modHandle document.DirHandle, dependsOn job.IDs) (job.IDs, error) {
	ids := make(job.IDs, 0)

	metaId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.LoadModuleMetadata(ctx, idx.modStore, modHandle.Path())
		},
		Type:      op.OpTypeLoadModuleMetadata.String(),
		DependsOn: dependsOn,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, metaId)

	refTargetsId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeReferenceTargets(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type:      op.OpTypeDecodeReferenceTargets.String(),
		DependsOn: job.IDs{metaId},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, refTargetsId)

	refOriginsId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeReferenceOrigins(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type:      op.OpTypeDecodeReferenceOrigins.String(),
		DependsOn: job.IDs{metaId},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, refOriginsId)

	registryId, err := idx.jobStore.EnqueueJob(job.Job{
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

	ids = append(ids, registryId)

	return ids, nil
}
