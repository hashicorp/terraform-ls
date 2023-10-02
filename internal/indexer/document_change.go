// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package indexer

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
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

	// Changes to a setting currently requires a LS restart, so the LS
	// setting context cannot change during the execution of a job. That's
	// why we can extract it here and use it in Defer.
	// See https://github.com/hashicorp/terraform-ls/issues/1008
	validationOptions, err := lsctx.ValidationOptions(ctx)
	if err != nil {
		return ids, err
	}

	metaId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.LoadModuleMetadata(ctx, idx.modStore, modHandle.Path())
		},
		Type:        op.OpTypeLoadModuleMetadata.String(),
		DependsOn:   dependsOn,
		IgnoreState: ignoreState,
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			ids := make(job.IDs, 0)
			if jobErr != nil {
				idx.logger.Printf("loading module metadata returned error: %s", jobErr)
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
				Type:        op.OpTypePreloadEmbeddedSchema.String(),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return ids, err
			}
			ids = append(ids, eSchemaId)

			if validationOptions.EnableEnhancedValidation {
				_, err = idx.jobStore.EnqueueJob(ctx, job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						return module.SchemaModuleValidation(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
					},
					Type:        op.OpTypeSchemaModuleValidation.String(),
					DependsOn:   append(modCalls, eSchemaId),
					IgnoreState: ignoreState,
				})
				if err != nil {
					return ids, err
				}
			}

			refTargetsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeReferenceTargets(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
				},
				Type:        op.OpTypeDecodeReferenceTargets.String(),
				DependsOn:   job.IDs{eSchemaId},
				IgnoreState: ignoreState,
			})
			if err != nil {
				return ids, err
			}
			ids = append(ids, refTargetsId)

			refOriginsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return module.DecodeReferenceOrigins(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
				},
				Type:        op.OpTypeDecodeReferenceOrigins.String(),
				DependsOn:   append(modCalls, eSchemaId),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return ids, err
			}
			ids = append(ids, refOriginsId)

			if validationOptions.EnableEnhancedValidation {
				_, err = idx.jobStore.EnqueueJob(ctx, job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						return module.ReferenceValidation(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
					},
					Type:        op.OpTypeReferenceValidation.String(),
					DependsOn:   job.IDs{refOriginsId, refTargetsId},
					IgnoreState: ignoreState,
				})
				if err != nil {
					return ids, err
				}
			}

			return ids, nil
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, metaId)

	// This job may make an HTTP request, and we schedule it in
	// the low-priority queue, so we don't want to wait for it.
	_, err = idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.GetModuleDataFromRegistry(ctx, idx.registryClient,
				idx.modStore, idx.registryModStore, modHandle.Path())
		},
		Priority:  job.LowPriority,
		DependsOn: job.IDs{metaId},
		Type:      op.OpTypeGetModuleDataFromRegistry.String(),
	})
	if err != nil {
		return ids, err
	}

	return ids, nil
}
