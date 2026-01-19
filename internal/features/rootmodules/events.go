// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package rootmodules

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/ast"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/jobs"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func (f *RootModulesFeature) discover(path string, files []string) error {
	rawUri := uri.FromPath(path)
	if uri, ok := datadir.ModuleUriFromDataDir(rawUri); ok {
		f.logger.Printf("discovered root module in %s", uri)
		dir := document.DirHandleFromURI(uri)
		err := f.Store.AddIfNotExists(dir.Path())
		if err != nil {
			return err
		}

		return nil
	}

	for _, file := range files {
		if ast.IsRootModuleFilename(file) {
			f.logger.Printf("discovered root module file in %s", path)

			err := f.Store.AddIfNotExists(path)
			if err != nil {
				return err
			}

			break
		}
	}

	return nil
}

func (f *RootModulesFeature) didOpen(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	// There is no dedicated language id for root module related files
	// so we rely on the walker to discover root modules and add them to the
	// store during walking.

	// Schedule jobs if state entry exists
	hasModuleRootRecord := f.Store.Exists(path)
	if !hasModuleRootRecord {
		return ids, nil
	}

	versionId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, f.tfExecFactory)
			return jobs.GetTerraformVersion(ctx, f.Store, path)
		},
		Type: op.OpTypeGetTerraformVersion.String(),
	})
	if err != nil {
		return ids, nil
	}
	ids = append(ids, versionId)

	modManifestId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseModuleManifest(ctx, f.fs, f.Store, dir.Path())
		},
		Type: op.OpTypeParseModuleManifest.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, modManifestId)

	terraformSourcesId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseTerraformSources(ctx, f.fs, f.Store, dir.Path())
		},
		Type: op.OpTypeParseTerraformSources.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, terraformSourcesId)

	pSchemaVerId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseProviderVersions(ctx, f.fs, f.Store, path)
		},
		Type: op.OpTypeParseProviderVersions.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, pSchemaVerId)

	pSchemaId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, f.tfExecFactory)
			return jobs.ObtainSchema(ctx, f.Store, f.stateStore.ProviderSchemas, path)
		},
		Type:      op.OpTypeObtainSchema.String(),
		DependsOn: job.IDs{pSchemaVerId},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, pSchemaId)

	return ids, nil
}

func (f *RootModulesFeature) pluginLockChange(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	// We might not have a record yet, so we add it
	err := f.Store.AddIfNotExists(path)
	if err != nil {
		return ids, err
	}

	pSchemaVerId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseProviderVersions(ctx, f.fs, f.Store, path)
		},
		IgnoreState: true,
		Type:        op.OpTypeParseProviderVersions.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, pSchemaVerId)

	pSchemaId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			ctx = exec.WithExecutorFactory(ctx, f.tfExecFactory)
			return jobs.ObtainSchema(ctx, f.Store, f.stateStore.ProviderSchemas, path)
		},
		IgnoreState: true,
		Type:        op.OpTypeObtainSchema.String(),
		DependsOn:   job.IDs{pSchemaVerId},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, pSchemaId)

	return ids, nil
}

func (f *RootModulesFeature) manifestChange(ctx context.Context, dir document.DirHandle, changeType protocol.FileChangeType) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	// We might not have a record yet, so we add it
	err := f.Store.AddIfNotExists(path)
	if err != nil {
		return ids, err
	}

	if changeType == protocol.Deleted {
		// Manifest is deleted, so we clear the manifest from the store
		f.Store.UpdateModManifest(path, nil, nil)
		// We also delete the Terraform Sources (if they exist), since delete changes can also happen if the
		// entire .terraform directory is deleted and there should only be either a manifest or terraform sources anyway
		f.Store.UpdateTerraformSources(path, nil, nil)
		return ids, nil
	}

	modManifestId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseModuleManifest(ctx, f.fs, f.Store, path)
		},
		Type: op.OpTypeParseModuleManifest.String(),
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			return f.indexInstalledModuleCalls(ctx, dir)
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, modManifestId)

	terraformSourcesId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseTerraformSources(ctx, f.fs, f.Store, dir.Path())
		},
		Type: op.OpTypeParseTerraformSources.String(),
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			return f.indexTerraformSourcesDirs(ctx, dir)
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, terraformSourcesId)

	return ids, nil
}

func (f *RootModulesFeature) indexInstalledModuleCalls(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	jobIds := make(job.IDs, 0)

	moduleCalls, err := f.Store.InstalledModuleCalls(dir.Path())
	if err != nil {
		return jobIds, err
	}

	for _, mc := range moduleCalls {
		mcHandle := document.DirHandleFromPath(mc.Path)
		f.stateStore.WalkerPaths.EnqueueDir(ctx, mcHandle)
	}

	return jobIds, nil
}

func (f *RootModulesFeature) indexTerraformSourcesDirs(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	jobIds := make(job.IDs, 0)

	for _, subDir := range f.Store.TerraformSourcesDirectories(dir.Path()) {
		dh := document.DirHandleFromPath(filepath.Join(dir.Path(), subDir))
		f.stateStore.WalkerPaths.EnqueueDir(ctx, dh)
	}

	return jobIds, nil
}
