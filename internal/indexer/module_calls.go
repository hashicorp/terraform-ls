// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package indexer

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfmodule "github.com/hashicorp/terraform-schema/module"
)

func (idx *Indexer) decodeInstalledModuleCalls(ctx context.Context, modHandle document.DirHandle, ignoreState bool) (job.IDs, error) {
	jobIds := make(job.IDs, 0)

	moduleCalls, err := idx.modStore.ModuleCalls(modHandle.Path())
	if err != nil {
		return jobIds, err
	}

	var errs *multierror.Error

	idx.logger.Printf("indexing installed module calls: %d", len(moduleCalls.Installed))
	for _, mc := range moduleCalls.Installed {
		fi, err := os.Stat(mc.Path)
		if err != nil || !fi.IsDir() {
			multierror.Append(errs, err)
			continue
		}
		err = idx.modStore.Add(mc.Path)
		if err != nil {
			multierror.Append(errs, err)
			continue
		}

		mcHandle := document.DirHandleFromPath(mc.Path)
		mcJobIds, mcErr := idx.decodeModuleAtPath(ctx, mcHandle, ignoreState)
		jobIds = append(jobIds, mcJobIds...)
		multierror.Append(errs, mcErr)
	}

	return jobIds, errs.ErrorOrNil()
}

func (idx *Indexer) decodeDeclaredModuleCalls(ctx context.Context, modHandle document.DirHandle, ignoreState bool) (job.IDs, error) {
	jobIds := make(job.IDs, 0)

	moduleCalls, err := idx.modStore.ModuleCalls(modHandle.Path())
	if err != nil {
		return jobIds, err
	}

	var errs *multierror.Error

	idx.logger.Printf("indexing declared module calls for %q: %d", modHandle.URI, len(moduleCalls.Declared))
	for _, mc := range moduleCalls.Declared {
		localSource, ok := mc.SourceAddr.(tfmodule.LocalSourceAddr)
		if !ok {
			continue
		}
		mcPath := filepath.Join(modHandle.Path(), filepath.FromSlash(localSource.String()))

		fi, err := os.Stat(mcPath)
		if err != nil || !fi.IsDir() {
			multierror.Append(errs, err)
			continue
		}

		mcIgnoreState := ignoreState
		err = idx.modStore.Add(mcPath)
		if err != nil {
			alreadyExistsErr := &state.AlreadyExistsError{}
			if errors.As(err, &alreadyExistsErr) {
				mcIgnoreState = false
			} else {
				multierror.Append(errs, err)
				continue
			}
		}

		mcHandle := document.DirHandleFromPath(mcPath)
		mcJobIds, mcErr := idx.decodeModuleAtPath(ctx, mcHandle, mcIgnoreState)
		jobIds = append(jobIds, mcJobIds...)
		multierror.Append(errs, mcErr)
	}

	return jobIds, errs.ErrorOrNil()
}

func (idx *Indexer) decodeModuleAtPath(ctx context.Context, modHandle document.DirHandle, ignoreState bool) (job.IDs, error) {
	var errs *multierror.Error
	jobIds := make(job.IDs, 0)
	refCollectionDeps := make(job.IDs, 0)

	parseId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleConfiguration(ctx, idx.fs, idx.modStore, modHandle.Path())
		},
		Type:        op.OpTypeParseModuleConfiguration.String(),
		IgnoreState: ignoreState,
	})
	if err != nil {
		multierror.Append(errs, err)
	} else {
		jobIds = append(jobIds, parseId)
		refCollectionDeps = append(refCollectionDeps, parseId)
	}

	var metaId job.ID
	if parseId != "" {
		metaId, err = idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir:  modHandle,
			Type: op.OpTypeLoadModuleMetadata.String(),
			Func: func(ctx context.Context) error {
				return module.LoadModuleMetadata(ctx, idx.modStore, modHandle.Path())
			},
			DependsOn:   job.IDs{parseId},
			IgnoreState: ignoreState,
		})
		if err != nil {
			multierror.Append(errs, err)
		} else {
			jobIds = append(jobIds, metaId)
			refCollectionDeps = append(refCollectionDeps, metaId)
		}

		eSchemaId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.PreloadEmbeddedSchema(ctx, idx.logger, schemas.FS, idx.modStore, idx.schemaStore, modHandle.Path())
			},
			Type:        op.OpTypePreloadEmbeddedSchema.String(),
			DependsOn:   job.IDs{metaId},
			IgnoreState: ignoreState,
		})
		if err != nil {
			multierror.Append(errs, err)
		} else {
			jobIds = append(jobIds, eSchemaId)
			refCollectionDeps = append(refCollectionDeps, eSchemaId)
		}
	}

	if parseId != "" {
		ids, err := idx.collectReferences(ctx, modHandle, refCollectionDeps, ignoreState)
		if err != nil {
			multierror.Append(errs, err)
		} else {
			jobIds = append(jobIds, ids...)
		}
	}

	varsParseId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseVariables(ctx, idx.fs, idx.modStore, modHandle.Path())
		},
		Type:        op.OpTypeParseVariables.String(),
		IgnoreState: ignoreState,
	})
	if err != nil {
		multierror.Append(errs, err)
	} else {
		jobIds = append(jobIds, varsParseId)
	}

	if varsParseId != "" {
		varsRefId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.DecodeVarsReferences(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
			},
			Type:        op.OpTypeDecodeVarsReferences.String(),
			DependsOn:   job.IDs{varsParseId},
			IgnoreState: ignoreState,
		})
		if err != nil {
			multierror.Append(errs, err)
		} else {
			jobIds = append(jobIds, varsRefId)
		}
	}

	return jobIds, errs.ErrorOrNil()
}

func (idx *Indexer) collectReferences(ctx context.Context, modHandle document.DirHandle, dependsOn job.IDs, ignoreState bool) (job.IDs, error) {
	ids := make(job.IDs, 0)

	var errs *multierror.Error

	id, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeReferenceTargets(ctx, idx.recordStores.Modules, idx.schemaStore, modHandle.Path())
		},
		Type:        op.OpTypeDecodeReferenceTargets.String(),
		DependsOn:   dependsOn,
		IgnoreState: ignoreState,
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, id)
	}

	id, err = idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeReferenceOrigins(ctx, idx.recordStores.Modules, idx.schemaStore, modHandle.Path())
		},
		Type:        op.OpTypeDecodeReferenceOrigins.String(),
		DependsOn:   dependsOn,
		IgnoreState: ignoreState,
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, id)
	}

	return ids, errs.ErrorOrNil()
}
