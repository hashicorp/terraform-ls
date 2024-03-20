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
	ids := make(job.IDs, 0)
	var errs *multierror.Error

	hasRootRecord := idx.recordStores.Roots.Exists(modHandle.Path())
	if hasRootRecord {
		_, err := idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
				return module.GetTerraformVersion(ctx, idx.recordStores.Roots, modHandle.Path())
			},
			Type: op.OpTypeGetTerraformVersion.String(),
		})
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		// Given that getting version may take time and we only use it to
		// enhance the UX, we ignore the outcome (job ID) here
		// to avoid delays when documents of new modules are open.

		dataDir := datadir.WalkDataDirOfModule(idx.fs, modHandle.Path())
		idx.logger.Printf("parsed datadir: %#v", dataDir)

	}

	// Work related to module files
	hasModuleRecord := idx.recordStores.Modules.Exists(modHandle.Path())
	moduleJobIds := make(job.IDs, 0)
	if hasModuleRecord {
		parseId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.ParseModuleConfiguration(ctx, idx.fs, idx.recordStores.Modules, modHandle.Path())
			},
			Type:        op.OpTypeParseModuleConfiguration.String(),
			IgnoreState: true,
		})
		if err != nil {
			return ids, err
		}
		moduleJobIds = append(moduleJobIds, parseId)

		metaId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.LoadModuleMetadata(ctx, idx.recordStores.Modules, modHandle.Path())
			},
			Type:        op.OpTypeLoadModuleMetadata.String(),
			DependsOn:   job.IDs{parseId},
			IgnoreState: true,
			Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
				deferIds := make(job.IDs, 0)
				if jobErr != nil {
					idx.logger.Printf("loading module metadata returned error: %s", jobErr)
				}

				modCalls, mcErr := idx.decodeDeclaredModuleCalls(ctx, modHandle, true)
				if mcErr != nil {
					idx.logger.Printf("decoding declared module calls for %q failed: %s", modHandle.URI, mcErr)
					// We log the error but still continue scheduling other jobs
					// which are still valuable for the rest of the configuration
					// even if they may not have the data for module calls.
				}

				eSchemaId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						return module.PreloadEmbeddedSchema(ctx, idx.logger, schemas.FS, idx.recordStores.Modules, idx.recordStores.ProviderSchemas, modHandle.Path())
					},
					Type:        op.OpTypePreloadEmbeddedSchema.String(),
					IgnoreState: true,
				})
				if err != nil {
					return deferIds, err
				}
				deferIds = append(deferIds, eSchemaId)

				refTargetsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						return module.DecodeReferenceTargets(ctx, idx.recordStores.Modules, idx.recordStores, modHandle.Path())
					},
					Type:        op.OpTypeDecodeReferenceTargets.String(),
					DependsOn:   job.IDs{eSchemaId},
					IgnoreState: true,
				})
				if err != nil {
					return deferIds, err
				}
				deferIds = append(deferIds, refTargetsId)

				refOriginsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						return module.DecodeReferenceOrigins(ctx, idx.recordStores.Modules, idx.recordStores, modHandle.Path())
					},
					Type:        op.OpTypeDecodeReferenceOrigins.String(),
					DependsOn:   append(modCalls, eSchemaId),
					IgnoreState: true,
				})
				if err != nil {
					return deferIds, err
				}
				deferIds = append(deferIds, refOriginsId)

				return deferIds, nil
			},
		})
		if err != nil {
			return ids, err
		}
		moduleJobIds = append(moduleJobIds, metaId)

		// This job may make an HTTP request, and we schedule it in
		// the low-priority queue, so we don't want to wait for it.
		_, err = idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.GetModuleDataFromRegistry(ctx, idx.registryClient,
					idx.recordStores.Modules, idx.recordStores.RegistryModules, modHandle.Path())
			},
			Priority:  job.LowPriority,
			DependsOn: job.IDs{metaId},
			Type:      op.OpTypeGetModuleDataFromRegistry.String(),
		})
		if err != nil {
			return ids, err
		}
	}
	ids = append(ids, moduleJobIds...)

	// Work related to variable definition files
	hasVariableRecord := idx.recordStores.Variables.Exists(modHandle.Path())
	variableJobIds := make(job.IDs, 0)
	if hasVariableRecord {
		parseVarsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.ParseVariables(ctx, idx.fs, idx.recordStores.Variables, modHandle.Path())
			},
			Type:        op.OpTypeParseVariables.String(),
			IgnoreState: true,
		})
		if err != nil {
			return ids, err
		}
		variableJobIds = append(variableJobIds, parseVarsId)

		varsRefsId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.DecodeVarsReferences(ctx, idx.recordStores.Variables, idx.recordStores, modHandle.Path())
			},
			Type:      op.OpTypeDecodeVarsReferences.String(),
			DependsOn: job.IDs{parseVarsId},
		})
		if err != nil {
			return ids, err
		}
		variableJobIds = append(variableJobIds, varsRefsId)
	}
	ids = append(ids, variableJobIds...)

	// Validation for the whole directory
	validationOptions, err := lsctx.ValidationOptions(ctx)
	if err != nil {
		return ids, err
	}

	if validationOptions.EnableEnhancedValidation {
		_, err = idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.SchemaModuleValidation(ctx, idx.recordStores.Modules, idx.recordStores, modHandle.Path())
			},
			Type:        op.OpTypeSchemaModuleValidation.String(),
			DependsOn:   moduleJobIds,
			IgnoreState: true,
		})
		if err != nil {
			return ids, err
		}

		_, err = idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.ReferenceValidation(ctx, idx.recordStores.Modules, idx.recordStores, modHandle.Path())
			},
			Type:        op.OpTypeReferenceValidation.String(),
			DependsOn:   moduleJobIds,
			IgnoreState: true,
		})
		if err != nil {
			return ids, err
		}

		_, err = idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.SchemaVariablesValidation(ctx, idx.recordStores.Variables, idx.recordStores, modHandle.Path())
			},
			Type:        op.OpTypeSchemaVarsValidation.String(),
			DependsOn:   append(moduleJobIds, variableJobIds...),
			IgnoreState: true,
		})
		if err != nil {
			return ids, err
		}
	}

	return ids, errs.ErrorOrNil()
}
