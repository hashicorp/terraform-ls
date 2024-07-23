// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package stacks

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/jobs"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/module"
)

func (f *StacksFeature) discover(path string, files []string) error {
	for _, file := range files {
		if globalAst.IsIgnoredFile(file) {
			continue
		}

		if ast.IsStackFilename(file) || ast.IsDeployFilename(file) {
			f.logger.Printf("discovered stack file in %s", path)

			err := f.store.AddIfNotExists(path)
			if err != nil {
				return err
			}

			break
		}
	}

	return nil
}

func (f *StacksFeature) didOpen(ctx context.Context, dir document.DirHandle, languageID string) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()
	f.logger.Printf("did open %q %q", path, languageID)

	// We need to decide if the path is relevant to us
	if languageID != lsp.Stacks.String() && languageID != lsp.Deploy.String() {
		return ids, nil
	}

	// Add to state as path is relevant
	err := f.store.AddIfNotExists(path)
	if err != nil {
		return ids, err
	}

	tfVersion, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.LoadTerraformVersion(ctx, f.fs, f.store, path)
		},
		Type: operation.OpTypeLoadStackRequiredTerraformVersion.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, tfVersion)

	decodeIds, err := f.decodeStack(ctx, dir, false, true)
	if err != nil {
		return ids, err
	}
	ids = append(ids, decodeIds...)

	return ids, err
}

func (f *StacksFeature) didChange(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	hasStackRecord := f.store.Exists(dir.Path())
	if !hasStackRecord {
		return job.IDs{}, nil
	}

	return f.decodeStack(ctx, dir, true, true)
}

func (f *StacksFeature) didChangeWatched(ctx context.Context, rawPath string, changeType protocol.FileChangeType, isDir bool) (job.IDs, error) {
	ids := make(job.IDs, 0)

	switch changeType {
	case protocol.Deleted:
		// We don't know whether file or dir is being deleted
		// 1st we just blindly try to look it up as a directory
		hasStackRecord := f.store.Exists(rawPath)
		if hasStackRecord {
			f.removeIndexedStack(rawPath)
			return ids, nil
		}

		// 2nd we try again assuming it is a file
		parentDir := filepath.Dir(rawPath)
		hasStackRecord = f.store.Exists(parentDir)
		if !hasStackRecord {
			// Nothing relevant found in the feature state
			return ids, nil
		}

		// and check the parent directory still exists
		fi, err := os.Stat(parentDir)
		if err != nil {
			if os.IsNotExist(err) {
				// if not, we remove the indexed module
				f.removeIndexedStack(rawPath)
				return ids, nil
			}
			f.logger.Printf("error checking existence (%q deleted): %s", parentDir, err)
			return ids, nil
		}
		if !fi.IsDir() {
			// Should never happen
			f.logger.Printf("error: %q (deleted) is not a directory", parentDir)
			return ids, nil
		}

		// If the parent directory exists, we just need to
		// check if the there are open documents for the path and the
		// path is a module path. If so, we need to reparse the module.
		dir := document.DirHandleFromPath(parentDir)
		hasOpenDocs, err := f.stateStore.DocumentStore.HasOpenDocuments(dir)
		if err != nil {
			f.logger.Printf("error when checking for open documents in path (%q deleted): %s", rawPath, err)
		}
		if !hasOpenDocs {
			return ids, nil
		}

		return f.decodeStack(ctx, dir, true, true)

	case protocol.Changed:
		fallthrough
	case protocol.Created:
		var dir document.DirHandle
		if isDir {
			dir = document.DirHandleFromPath(rawPath)
		} else {
			docHandle := document.HandleFromPath(rawPath)
			dir = docHandle.Dir
		}

		// Check if the there are open documents for the path and the
		// path is a module path. If so, we need to reparse the module.
		hasOpenDocs, err := f.stateStore.DocumentStore.HasOpenDocuments(dir)
		if err != nil {
			f.logger.Printf("error when checking for open documents in path (%q changed): %s", rawPath, err)
		}
		if !hasOpenDocs {
			return ids, nil
		}

		hasModuleRecord := f.store.Exists(dir.Path())
		if !hasModuleRecord {
			return ids, nil
		}

		return f.decodeStack(ctx, dir, true, true)
	}

	return nil, nil
}

func (f *StacksFeature) decodeStack(ctx context.Context, dir document.DirHandle, ignoreState bool, isFirstLevel bool) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	parseId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseStackConfiguration(ctx, f.fs, f.store, path)
		},
		Type:        operation.OpTypeParseStackConfiguration.String(),
		IgnoreState: ignoreState,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseId)

	metaId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.LoadStackMetadata(ctx, f.store, path)
		},
		Type:        operation.OpTypeLoadStackMetadata.String(),
		DependsOn:   job.IDs{parseId},
		IgnoreState: ignoreState,
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			if jobErr != nil {
				f.logger.Printf("loading module metadata returned error: %s", jobErr)
			}

			spawnedIds, err := loadStackComponentSources(ctx, f.store, f.bus, path)
			if err != nil {
				return spawnedIds, err
			}

			// while we now have the job ids in here, depending on the metaId job is not enough
			// to await these spawned jobs, so we will need to move all depending jobs to this function
			// as well. e.g. LoadStackComponentSources, PreloadEmbeddedSchema (because future ref collection jobs depend on it), etc.
			// we might just move all in here for simplicity

			// Reference collection jobs will depend on this one, so we move it here in advance
			eSchemaId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.PreloadEmbeddedSchema(ctx, f.logger, schemas.FS,
						f.store, f.stateStore.ProviderSchemas, path)
				},
				// DependsOn: none required, since we are inside
				Type:        operation.OpTypeStacksPreloadEmbeddedSchema.String(),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return spawnedIds, err
			}
			spawnedIds = append(spawnedIds, eSchemaId)

			return spawnedIds, nil
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, metaId)

	// TODO: Implement the following functions where appropriate to stacks
	// Future: decodeDeclaredModuleCalls(ctx, dir, ignoreState)
	// Future: DecodeReferenceTargets(ctx, f.Store, f.rootFeature, path)
	// Future: DecodeReferenceOrigins(ctx, f.Store, f.rootFeature, path)

	return ids, nil
}

func (f *StacksFeature) removeIndexedStack(rawPath string) {
	stackHandle := document.DirHandleFromPath(rawPath)

	err := f.stateStore.JobStore.DequeueJobsForDir(stackHandle)
	if err != nil {
		f.logger.Printf("failed to dequeue jobs for stack: %s", err)
		return
	}

	err = f.store.Remove(rawPath)
	if err != nil {
		f.logger.Printf("failed to remove stack from state: %s", err)
		return
	}
}

// loadStackComponentSources will trigger parsing the local terraform modules for a stack in the ModulesFeature
func loadStackComponentSources(ctx context.Context, stackStore *state.StackStore, bus *eventbus.EventBus, stackPath string) (job.IDs, error) {
	ids := make(job.IDs, 0)

	record, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return ids, err
	}

	// iterate over each component in the stack and find local terraform modules
	for _, component := range record.Meta.Components {
		if component.Source == "" {
			// no source recorded, skip
			continue
		}

		var fullPath string
		// detect if component.Source is a local module
		switch component.SourceAddr.(type) {
		case module.LocalSourceAddr:
			fullPath = filepath.Join(stackPath, filepath.FromSlash(component.Source))
		case tfaddr.Module:
			continue
		case module.RemoteSourceAddr:
			continue
		default:
			// Unknown source address, we can't resolve the path
			continue
		}

		if fullPath == "" {
			// Unknown source address, we can't resolve the path
			continue
		}

		dh := document.DirHandleFromPath(fullPath)

		// notify the event bus that a new Component with a
		// local modules has been opened
		spawnedIds := bus.DidOpen(eventbus.DidOpenEvent{
			Context:    ctx,
			Dir:        dh,
			LanguageID: lsp.Terraform.String(),
		})

		ids = append(ids, spawnedIds...)
	}

	return ids, nil
}
