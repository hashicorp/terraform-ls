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

	// blockingJobIds tracks job IDs which need to finish
	// prior to collecting references
	blockingJobIds := make(job.IDs, 0)

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleConfiguration(idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseModuleConfiguration.String(),
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			ids := make(job.IDs, 0)

			id, err := idx.jobStore.EnqueueJob(job.Job{
				Dir:  modHandle,
				Type: op.OpTypeLoadModuleMetadata.String(),
				Func: func(ctx context.Context) error {
					return module.LoadModuleMetadata(idx.modStore, modHandle.Path())
				},
				Defer: idx.decodeDeclaredModuleCalls(modHandle),
			})
			if err != nil {
				return ids, err
			} else {
				ids = append(ids, id)
			}

			return ids, nil
		},
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, id)
		blockingJobIds = append(blockingJobIds, id)
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
		blockingJobIds = append(blockingJobIds, id)
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
			blockingJobIds = append(blockingJobIds, id)
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
			blockingJobIds = append(blockingJobIds, id)
		}
	}

	// Here we wait for all dependent jobs to be processed to
	// reflect any data required to collect reference origins.
	// This assumes scheduler is running to consume the jobs
	// by the time we reach this point.
	err = idx.jobStore.WaitForJobs(ctx, blockingJobIds...)
	if err != nil {
		return ids, err
	}

	rIds, err := idx.collectReferences(ctx, modHandle)
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, rIds...)
	}

	return ids, errs.ErrorOrNil()
}
