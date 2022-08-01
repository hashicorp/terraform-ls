package indexer

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (idx *Indexer) WalkedModule(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)
	var errs *multierror.Error

	parseId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleConfiguration(idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseModuleConfiguration.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, parseId)
	}

	metaId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir:  modHandle,
		Type: op.OpTypeLoadModuleMetadata.String(),
		Func: func(ctx context.Context) error {
			return module.LoadModuleMetadata(idx.modStore, modHandle.Path())
		},
		DependsOn: job.IDs{parseId},
	})
	if err != nil {
		return ids, err
	} else {
		ids = append(ids, metaId)
	}

	parseVarsId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseVariables(idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseVariables.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, parseVarsId)
	}

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

	tfVersionId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
			return module.GetTerraformVersion(ctx, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeGetTerraformVersion.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, tfVersionId)
	}

	dataDir := datadir.WalkDataDirOfModule(idx.fs, modHandle.Path())
	idx.logger.Printf("parsed datadir: %#v", dataDir)

	refCollectionDeps := job.IDs{
		parseId, metaId, tfVersionId,
	}
	if dataDir.PluginLockFilePath != "" {
		pSchemaId, err := idx.jobStore.EnqueueJob(job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
				return module.ObtainSchema(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
			},
			Type: op.OpTypeObtainSchema.String(),
		})
		if err != nil {
			errs = multierror.Append(errs, err)
		} else {
			ids = append(ids, pSchemaId)
			refCollectionDeps = append(refCollectionDeps, pSchemaId)
		}
	}

	if dataDir.ModuleManifestPath != "" {
		// References are collected *after* manifest parsing
		// so that we reflect any references to submodules.
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
			errs = multierror.Append(errs, err)
		} else {
			ids = append(ids, modManifestId)
			refCollectionDeps = append(refCollectionDeps, modManifestId)
		}
	}

	rIds, err := idx.collectReferences(modHandle, refCollectionDeps)
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, rIds...)
	}

	return ids, errs.ErrorOrNil()
}
