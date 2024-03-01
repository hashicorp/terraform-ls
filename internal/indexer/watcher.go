// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

	modManifestId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleManifest(ctx, idx.fs, idx.recordStores.Modules, modHandle.Path())
		},
		Type:        op.OpTypeParseModuleManifest.String(),
		IgnoreState: true,
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			return idx.decodeInstalledModuleCalls(ctx, modHandle, true)
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
	dependsOn := make(job.IDs, 0)
	var errs *multierror.Error

	pSchemaVerId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseProviderVersions(ctx, idx.fs, idx.recordStores.Modules, modHandle.Path())
		},
		IgnoreState: true,
		Type:        op.OpTypeParseProviderVersions.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, pSchemaVerId)
		dependsOn = append(dependsOn, pSchemaVerId)
	}

	pSchemaId, err := idx.jobStore.EnqueueJob(ctx, job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
			return module.ObtainSchema(ctx, idx.recordStores.Modules, idx.schemaStore, modHandle.Path())
		},
		IgnoreState: true,
		Type:        op.OpTypeObtainSchema.String(),
		DependsOn:   dependsOn,
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, pSchemaId)
	}

	return ids, errs.ErrorOrNil()
}
