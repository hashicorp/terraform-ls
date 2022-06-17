package module

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

type Indexer struct {
	fs            ReadOnlyFS
	modStore      *state.ModuleStore
	schemaStore   *state.ProviderSchemaStore
	jobStore      job.JobStore
	tfExecFactory exec.ExecutorFactory
}

func NewIndexer(fs ReadOnlyFS, modStore *state.ModuleStore, schemaStore *state.ProviderSchemaStore,
	jobStore job.JobStore, tfExec exec.ExecutorFactory) *Indexer {
	return &Indexer{
		fs:            fs,
		modStore:      modStore,
		schemaStore:   schemaStore,
		jobStore:      jobStore,
		tfExecFactory: tfExec,
	}
}

func (idx *Indexer) ModuleManifestChanged(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return ParseModuleManifest(idx.fs, idx.modStore, modHandle.Path())
		},
		Type:  op.OpTypeParseModuleManifest.String(),
		Defer: decodeInstalledModuleCalls(idx.fs, idx.modStore, idx.schemaStore, modHandle.Path()),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	return ids, nil
}

func (idx *Indexer) PluginLockChanged(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)

	id, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
			eo, ok := exec.ExecutorOptsFromContext(ctx)
			if ok {
				ctx = exec.WithExecutorOpts(ctx, eo)
			}

			return ObtainSchema(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type: op.OpTypeObtainSchema.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	id, err = idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
			eo, ok := exec.ExecutorOptsFromContext(ctx)
			if ok {
				ctx = exec.WithExecutorOpts(ctx, eo)
			}

			return GetTerraformVersion(ctx, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeGetTerraformVersion.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, id)

	return ids, nil
}

func decodeInstalledModuleCalls(fs ReadOnlyFS, modStore *state.ModuleStore, schemaReader state.SchemaReader, modPath string) job.DeferFunc {
	return func(ctx context.Context, opErr error) (jobIds job.IDs) {
		if opErr != nil {
			return
		}

		moduleCalls, err := modStore.ModuleCalls(modPath)
		if err != nil {
			return
		}

		jobStore, err := job.JobStoreFromContext(ctx)
		if err != nil {
			return
		}

		for _, mc := range moduleCalls.Installed {
			fi, err := os.Stat(mc.Path)
			if err != nil || !fi.IsDir() {
				continue
			}
			modStore.Add(mc.Path)

			mcHandle := document.DirHandleFromPath(mc.Path)
			// copy path for queued jobs below
			mcPath := mc.Path

			id, err := jobStore.EnqueueJob(job.Job{
				Dir: mcHandle,
				Func: func(ctx context.Context) error {
					return ParseModuleConfiguration(fs, modStore, mcPath)
				},
				Type: op.OpTypeParseModuleConfiguration.String(),
				Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
					id, err := jobStore.EnqueueJob(job.Job{
						Dir:  mcHandle,
						Type: op.OpTypeLoadModuleMetadata.String(),
						Func: func(ctx context.Context) error {
							return LoadModuleMetadata(modStore, mcPath)
						},
					})
					if err != nil {
						return
					}
					ids = append(ids, id)

					rIds := collectReferences(ctx, mcHandle, modStore, schemaReader)
					ids = append(ids, rIds...)

					return
				},
			})
			if err != nil {
				return
			}
			jobIds = append(jobIds, id)

			id, err = jobStore.EnqueueJob(job.Job{
				Dir: mcHandle,
				Func: func(ctx context.Context) error {
					return ParseVariables(fs, modStore, mcPath)
				},
				Type: op.OpTypeParseVariables.String(),
				Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
					id, err = jobStore.EnqueueJob(job.Job{
						Dir: mcHandle,
						Func: func(ctx context.Context) error {
							return DecodeVarsReferences(ctx, modStore, schemaReader, mcPath)
						},
						Type: op.OpTypeDecodeVarsReferences.String(),
					})
					if err != nil {
						return
					}
					ids = append(ids, id)
					return
				},
			})
			if err != nil {
				return
			}
			jobIds = append(jobIds, id)
		}

		return
	}
}

func collectReferences(ctx context.Context, dirHandle document.DirHandle, modStore *state.ModuleStore, schemaReader state.SchemaReader) (ids job.IDs) {
	jobStore, err := job.JobStoreFromContext(ctx)
	if err != nil {
		return
	}

	id, err := jobStore.EnqueueJob(job.Job{
		Dir: dirHandle,
		Func: func(ctx context.Context) error {
			return DecodeReferenceTargets(ctx, modStore, schemaReader, dirHandle.Path())
		},
		Type: op.OpTypeDecodeReferenceTargets.String(),
	})
	if err != nil {
		return
	}
	ids = append(ids, id)

	id, err = jobStore.EnqueueJob(job.Job{
		Dir: dirHandle,
		Func: func(ctx context.Context) error {
			return DecodeReferenceOrigins(ctx, modStore, schemaReader, dirHandle.Path())
		},
		Type: op.OpTypeDecodeReferenceOrigins.String(),
	})
	if err != nil {
		return
	}
	ids = append(ids, id)

	return
}
