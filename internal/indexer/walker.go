package indexer

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (idx *Indexer) WalkedModule(ctx context.Context, modHandle document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)
	var errs *multierror.Error

	refCollectionDeps := make(job.IDs, 0)
	providerVersionDeps := make(job.IDs, 0)

	parseId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseModuleConfiguration(ctx, idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseModuleConfiguration.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, parseId)
		refCollectionDeps = append(refCollectionDeps, parseId)
		providerVersionDeps = append(providerVersionDeps, parseId)
	}

	var metaId job.ID
	if parseId != "" {
		metaId, err = idx.jobStore.EnqueueJob(job.Job{
			Dir:  modHandle,
			Type: op.OpTypeLoadModuleMetadata.String(),
			Func: func(ctx context.Context) error {
				return module.LoadModuleMetadata(ctx, idx.modStore, modHandle.Path())
			},
			DependsOn: job.IDs{parseId},
		})
		if err != nil {
			return ids, err
		} else {
			ids = append(ids, metaId)
			refCollectionDeps = append(refCollectionDeps, metaId)
			providerVersionDeps = append(providerVersionDeps, metaId)
		}
	}

	parseVarsId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.ParseVariables(ctx, idx.fs, idx.modStore, modHandle.Path())
		},
		Type: op.OpTypeParseVariables.String(),
	})
	if err != nil {
		errs = multierror.Append(errs, err)
	} else {
		ids = append(ids, parseVarsId)
	}

	if parseVarsId != "" {
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
		} else {
			ids = append(ids, varsRefsId)
			refCollectionDeps = append(refCollectionDeps, varsRefsId)
		}
	}

	dataDir := datadir.WalkDataDirOfModule(idx.fs, modHandle.Path())
	idx.logger.Printf("parsed datadir: %#v", dataDir)

	var modManifestId job.ID
	if dataDir.ModuleManifestPath != "" {
		// References are collected *after* manifest parsing
		// so that we reflect any references to submodules.
		modManifestId, err = idx.jobStore.EnqueueJob(job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.ParseModuleManifest(ctx, idx.fs, idx.modStore, modHandle.Path())
			},
			Type: op.OpTypeParseModuleManifest.String(),
			Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
				return idx.decodeInstalledModuleCalls(modHandle, false)
			},
		})
		if err != nil {
			errs = multierror.Append(errs, err)
		} else {
			ids = append(ids, modManifestId)
			refCollectionDeps = append(refCollectionDeps, modManifestId)
			// provider requirements may be within the (installed) modules
			providerVersionDeps = append(providerVersionDeps, modManifestId)
		}
	}

	if dataDir.PluginLockFilePath != "" {
		dependsOn := make(job.IDs, 0)
		pSchemaVerId, err := idx.jobStore.EnqueueJob(job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				return module.ParseProviderVersions(ctx, idx.fs, idx.modStore, modHandle.Path())
			},
			Type:      op.OpTypeParseProviderVersions.String(),
			DependsOn: providerVersionDeps,
		})
		if err != nil {
			errs = multierror.Append(errs, err)
		} else {
			ids = append(ids, pSchemaVerId)
			dependsOn = append(dependsOn, pSchemaVerId)
			refCollectionDeps = append(refCollectionDeps, pSchemaVerId)
		}

		pSchemaId, err := idx.jobStore.EnqueueJob(job.Job{
			Dir: modHandle,
			Func: func(ctx context.Context) error {
				ctx = exec.WithExecutorFactory(ctx, idx.tfExecFactory)
				return module.ObtainSchema(ctx, idx.modStore, idx.schemaStore, modHandle.Path())
			},
			Type:      op.OpTypeObtainSchema.String(),
			DependsOn: dependsOn,
		})
		if err != nil {
			errs = multierror.Append(errs, err)
		} else {
			ids = append(ids, pSchemaId)
			refCollectionDeps = append(refCollectionDeps, pSchemaId)
		}
	}

	eSchemaId, err := idx.jobStore.EnqueueJob(job.Job{
		Dir: modHandle,
		Func: func(ctx context.Context) error {
			return module.PreloadEmbeddedSchema(ctx, idx.logger, schemas.FS, idx.modStore, idx.schemaStore, modHandle.Path())
		},
		// This could theoretically also depend on ObtainSchema to avoid
		// attempt to preload the same schema twice but we avoid that dependency
		// as obtaining schema via CLI often takes a long time (multiple
		// seconds) and this would then defeat the main benefit
		// of preloaded schemas which can be loaded in miliseconds.
		DependsOn: providerVersionDeps,
		Type:      op.OpTypePreloadEmbeddedSchema.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, eSchemaId)

	if parseId != "" {
		rIds, err := idx.collectReferences(modHandle, refCollectionDeps, false)
		if err != nil {
			errs = multierror.Append(errs, err)
		} else {
			ids = append(ids, rIds...)
		}
	}

	return ids, errs.ErrorOrNil()
}
