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

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleConfiguration(idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseModuleConfiguration.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, id)
	}

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
			return ids, err
		},
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, id)
	}

	id, err = idx.jobStore.EnqueueJob(job.Job{
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
		ids = append(ids, id)
	}

	dataDir := datadir.WalkDataDirOfModule(idx.fs, modHandle.Path())
	idx.logger.Printf("parsed datadir: %#v", dataDir)

	if dataDir.PluginLockFilePath != "" {
		id, err := idx.jobStore.EnqueueJob(job.Job{
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
			ids = append(ids, id)
		}
	}

	if dataDir.ModuleManifestPath != "" {
		// References are collected *after* manifest parsing
		// so that we reflect any references to submodules.
		id, err := idx.jobStore.EnqueueJob(job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.ParseModuleManifest(idx.fs, idx.modStore, modHandle.Path())
			},
			Type:  op.OpTypeParseModuleManifest.String(),
			Defer: idx.decodeInstalledModuleCalls(modHandle),
		})
		if err != nil {
			errs = multierror.Append(errs, err)
		} else {
			ids = append(ids, id)
		}

		// Here we wait for all module calls to be processed to
		// reflect any metadata required to collect reference origins.
		// This assumes scheduler is running to consume the jobs
		// by the time we reach this point.
		idx.jobStore.WaitForJobs(ctx, id)
	}

	rIds, err := idx.collectReferences(ctx, modHandle)
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, rIds...)
	}

	return ids, errs.ErrorOrNil()
}
