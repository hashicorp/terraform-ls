package indexer

import (
	"context"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (idx *Indexer) decodeInstalledModuleCalls(modHandle document.DirHandle) job.DeferFunc {
	return func(ctx context.Context, opErr error) (job.IDs, error) {
		jobIds := make(job.IDs, 0)
		if opErr != nil {
			return jobIds, opErr
		}

		moduleCalls, err := idx.modStore.ModuleCalls(modHandle.Path())
		if err != nil {
			return jobIds, err
		}

		var errs *multierror.Error

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
			// copy path for queued jobs below
			mcPath := mc.Path

			id, err := idx.jobStore.EnqueueJob(job.Job{
				Dir: mcHandle,
				Func: func(ctx context.Context) error {
					return module.ParseModuleConfiguration(idx.fs, idx.modStore, mcPath)
				},
				Type: op.OpTypeParseModuleConfiguration.String(),
				Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
					ids := make(job.IDs, 0)

					id, err := idx.jobStore.EnqueueJob(job.Job{
						Dir:  mcHandle,
						Type: op.OpTypeLoadModuleMetadata.String(),
						Func: func(ctx context.Context) error {
							return module.LoadModuleMetadata(idx.modStore, mcPath)
						},
						Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
							return idx.collectReferences(ctx, mcHandle)
						},
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
				multierror.Append(errs, err)
				continue
			}
			jobIds = append(jobIds, id)

			id, err = idx.jobStore.EnqueueJob(job.Job{
				Dir: mcHandle,
				Func: func(ctx context.Context) error {
					return module.ParseVariables(idx.fs, idx.modStore, mcPath)
				},
				Type: op.OpTypeParseVariables.String(),
				Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
					ids := make(job.IDs, 0)
					id, err = idx.jobStore.EnqueueJob(job.Job{
						Dir: mcHandle,
						Func: func(ctx context.Context) error {
							return module.DecodeVarsReferences(ctx, idx.modStore, idx.schemaStore, mcPath)
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
				multierror.Append(errs, err)
				continue
			}
			jobIds = append(jobIds, id)
		}

		return jobIds, errs.ErrorOrNil()
	}
}

func (idx *Indexer) collectReferences(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	var errs *multierror.Error

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeReferenceTargets(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type: op.OpTypeDecodeReferenceTargets.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, id)
	}

	id, err = idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeReferenceOrigins(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type: op.OpTypeDecodeReferenceOrigins.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, id)
	}

	return ids, errs.ErrorOrNil()
}
