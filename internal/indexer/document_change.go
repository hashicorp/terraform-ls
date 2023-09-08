// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package indexer

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (idx *Indexer) DocumentChanged(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

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

	varsRefsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeVarsReferences(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type:        op.OpTypeDecodeVarsReferences.String(),
		DependsOn:   job.IDs{parseVarsId},
		IgnoreState: true,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, varsRefsId)

	return ids, nil
}

func (idx *Indexer) decodeModule(ctx context.Context, modHandle document.DirHandle, dependsOn job.IDs, ignoreState bool) (job.IDs, error) {
	ids := make(job.IDs, 0)

	metaId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.LoadModuleMetadata(ctx, idx.modStore, modHandle.Path())
		},
		Type:        op.OpTypeLoadModuleMetadata.String(),
		DependsOn:   dependsOn,
		IgnoreState: ignoreState,
		Defer: func(ctx context.Context, jobErr error) (jobIds job.IDs, err error) {
			if jobErr != nil {
				err = jobErr
				return
			}
			modCalls, mcErr := idx.decodeDeclaredModuleCalls(ctx, modHandle, ignoreState)
			if mcErr != nil {
				idx.logger.Printf("decoding declared module calls for %q failed: %s", modHandle.URI, mcErr)
				// We log the error but still continue scheduling other jobs
				// which are still valuable for the rest of the configuration
				// even if they may not have the data for module calls.
			}

			eSchemaId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.PreloadEmbeddedSchema(ctx, idx.logger, schemas.FS, idx.modStore, idx.schemaStore, modHandle.Path())
				},
				DependsOn:   modCalls,
				Type:        op.OpTypePreloadEmbeddedSchema.String(),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return
			}
			jobIds = append(jobIds, eSchemaId)

			refOriginsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeReferenceOrigins(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
				},
				Type:        op.OpTypeDecodeReferenceOrigins.String(),
				DependsOn:   append(modCalls, eSchemaId),
				IgnoreState: ignoreState,
			})
			jobIds = append(jobIds, refOriginsId)
			return
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, metaId)

	refTargetsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeReferenceTargets(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type:        op.OpTypeDecodeReferenceTargets.String(),
		DependsOn:   job.IDs{metaId},
		IgnoreState: ignoreState,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, refTargetsId)

	// This job may make an HTTP request, and we schedule it in
	// the low-priority queue, so we don't want to wait for it.
	_, err = idx.jobStore.EnqueueJob(ctx, job.Job{
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

	return ids, nil
}
