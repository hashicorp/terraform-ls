package indexer

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

type Indexer struct {
	logger           *log.Logger
	fs               ReadOnlyFS
	modStore         *state.ModuleStore
	schemaStore      *state.ProviderSchemaStore
	registryModStore *state.RegistryModuleStore
	jobStore         job.JobStore
	tfExecFactory    exec.ExecutorFactory
	registryClient   registry.Client
}

func NewIndexer(fs ReadOnlyFS, modStore *state.ModuleStore, schemaStore *state.ProviderSchemaStore,
	registryModStore *state.RegistryModuleStore, jobStore job.JobStore,
	tfExec exec.ExecutorFactory, registryClient registry.Client) *Indexer {

	discardLogger := log.New(ioutil.Discard, "", 0)

	return &Indexer{
		fs:               fs,
		modStore:         modStore,
		schemaStore:      schemaStore,
		registryModStore: registryModStore,
		jobStore:         jobStore,
		tfExecFactory:    tfExec,
		registryClient:   registryClient,
		logger:           discardLogger,
	}
}

func (idx *Indexer) SetLogger(logger *log.Logger) {
	idx.logger = logger
}

type Collector interface {
	CollectJobId(jobId job.ID)
}

func (idx *Indexer) decodeInstalledModuleCalls(modHandle document.DirHandle) job.DeferFunc {
	return func(ctx context.Context, opErr error) (jobIds job.IDs) {
		if opErr != nil {
			return
		}

		moduleCalls, err := idx.modStore.ModuleCalls(modHandle.Path())
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
			idx.modStore.Add(mc.Path)

			mcHandle := document.DirHandleFromPath(mc.Path)
			// copy path for queued jobs below
			mcPath := mc.Path

			id, err := jobStore.EnqueueJob(job.Job{
				Dir: mcHandle,
				Func: func(ctx context.Context) error {
					return module.ParseModuleConfiguration(idx.fs, idx.modStore, mcPath)
				},
				Type: op.OpTypeParseModuleConfiguration.String(),
				Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
					id, err := jobStore.EnqueueJob(job.Job{
						Dir:  mcHandle,
						Type: op.OpTypeLoadModuleMetadata.String(),
						Func: func(ctx context.Context) error {
							return module.LoadModuleMetadata(idx.modStore, mcPath)
						},
					})
					if err != nil {
						return
					}
					ids = append(ids, id)

					rIds := idx.collectReferences(ctx, mcHandle)
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
					return module.ParseVariables(idx.fs, idx.modStore, mcPath)
				},
				Type: op.OpTypeParseVariables.String(),
				Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
					id, err = jobStore.EnqueueJob(job.Job{
						Dir: mcHandle,
						Func: func(ctx context.Context) error {
							return module.DecodeVarsReferences(ctx, idx.modStore, idx.schemaStore, mcPath)
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

func (idx *Indexer) collectReferences(ctx context.Context, modHandle document.DirHandle) (ids job.IDs) {
	jobStore, err := job.JobStoreFromContext(ctx)
	if err != nil {
		return
	}

	id, err := jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeReferenceTargets(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type: op.OpTypeDecodeReferenceTargets.String(),
	})
	if err != nil {
		return
	}
	ids = append(ids, id)

	id, err = jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.DecodeReferenceOrigins(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		Type: op.OpTypeDecodeReferenceOrigins.String(),
	})
	if err != nil {
		return
	}
	ids = append(ids, id)

	return
}
