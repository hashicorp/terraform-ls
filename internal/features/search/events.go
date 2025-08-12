// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package search

import (
	"context"
	"os"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/search/ast"
	"github.com/hashicorp/terraform-ls/internal/features/search/jobs"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (f *SearchFeature) discover(path string, files []string) error {
	for _, file := range files {
		if globalAst.IsIgnoredFile(file) {
			continue
		}

		if ast.IsSearchFilename(file) {
			f.logger.Printf("discovered search file in %s", path)

			err := f.store.AddIfNotExists(path)
			if err != nil {
				return err
			}

			break
		}
	}

	return nil
}

func (f *SearchFeature) didOpen(ctx context.Context, dir document.DirHandle, languageID string) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()
	f.logger.Printf("did open %q %q", path, languageID)

	// We need to decide if the path is relevant to us
	if languageID != lsp.Search.String() {
		return ids, nil
	}

	// Add to state as path is relevant
	err := f.store.AddIfNotExists(path)
	if err != nil {
		return ids, err
	}

	decodeIds, err := f.decodeSearch(ctx, dir, false, true)
	if err != nil {
		return ids, err
	}
	ids = append(ids, decodeIds...)

	return ids, err
}

func (f *SearchFeature) didChange(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	hasSearchRecord := f.store.Exists(dir.Path())
	if !hasSearchRecord {
		return job.IDs{}, nil
	}

	return f.decodeSearch(ctx, dir, true, true)
}

func (f *SearchFeature) didChangeWatched(ctx context.Context, rawPath string, changeType protocol.FileChangeType, isDir bool) (job.IDs, error) {
	ids := make(job.IDs, 0)

	switch changeType {
	case protocol.Deleted:
		// We don't know whether file or dir is being deleted
		// 1st we just blindly try to look it up as a directory
		hasSearchRecord := f.store.Exists(rawPath)
		if hasSearchRecord {
			f.removeIndexedSearch(rawPath)
			return ids, nil
		}

		// 2nd we try again assuming it is a file
		parentDir := filepath.Dir(rawPath)
		hasSearchRecord = f.store.Exists(parentDir)
		if !hasSearchRecord {
			// Nothing relevant found in the feature state
			return ids, nil
		}

		// and check the parent directory still exists
		fi, err := os.Stat(parentDir)
		if err != nil {
			if os.IsNotExist(err) {
				// if not, we remove the indexed module
				f.removeIndexedSearch(rawPath)
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

		return f.decodeSearch(ctx, dir, true, true)

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

		return f.decodeSearch(ctx, dir, true, true)
	}

	return nil, nil
}

func (f *SearchFeature) decodeSearch(ctx context.Context, dir document.DirHandle, ignoreState bool, isFirstLevel bool) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	parseId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseSearchConfiguration(ctx, f.fs, f.store, path)
		},
		Type:        operation.OpTypeParseSearchConfiguration.String(),
		IgnoreState: ignoreState,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseId)

	// Changes to a setting currently requires a LS restart, so the LS
	// setting context cannot change during the execution of a job. That's
	// why we can extract it here and use it in Defer.
	// See https://github.com/hashicorp/terraform-ls/issues/1008
	// We can safely ignore the error here. If we can't get the options from
	// the context, validationOptions.EnableEnhancedValidation will be false
	// by default. So we don't run the validation jobs.
	validationOptions, _ := lsctx.ValidationOptions(ctx)

	metaId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.LoadSearchMetadata(ctx, f.store, f.moduleFeature, f.logger, path)
		},
		Type:        operation.OpTypeLoadSearchMetadata.String(),
		DependsOn:   job.IDs{parseId},
		IgnoreState: ignoreState,
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			deferIds := make(job.IDs, 0)

			if jobErr != nil {
				f.logger.Printf("loading module metadata returned error: %s", jobErr)
			}

			// Reference collection jobs will depend on this one, so we move it here in advance
			eSchemaId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.PreloadEmbeddedSchema(ctx, f.logger, schemas.FS,
						f.store, f.stateStore.ProviderSchemas, path)
				},
				Type:        operation.OpTypeSearchPreloadEmbeddedSchema.String(),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, eSchemaId)

			refTargetsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.DecodeReferenceTargets(ctx, f.store, f.moduleFeature, f.rootFeature, path)
				},
				Type:        operation.OpTypeDecodeReferenceTargets.String(),
				DependsOn:   job.IDs{eSchemaId},
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, refTargetsId)

			refOriginsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.DecodeReferenceOrigins(ctx, f.store, f.moduleFeature, f.rootFeature, path)
				},
				Type:        operation.OpTypeDecodeReferenceOrigins.String(),
				DependsOn:   job.IDs{eSchemaId},
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, refOriginsId)

			if validationOptions.EnableEnhancedValidation {
				_, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
					Dir: dir,
					Func: func(ctx context.Context) error {
						return jobs.SchemaSearchValidation(ctx, f.store, f.moduleFeature, f.rootFeature, dir.Path())
					},
					Type:        operation.OpTypeSchemaSearchValidation.String(),
					DependsOn:   job.IDs{refOriginsId, refTargetsId},
					IgnoreState: ignoreState,
				})
				if err != nil {
					return ids, err
				}
			}

			return deferIds, nil
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, metaId)

	return ids, nil
}

func (f *SearchFeature) removeIndexedSearch(rawPath string) {
	searchandle := document.DirHandleFromPath(rawPath)

	err := f.stateStore.JobStore.DequeueJobsForDir(searchandle)
	if err != nil {
		f.logger.Printf("failed to dequeue jobs for search: %s", err)
		return
	}

	err = f.store.Remove(rawPath)
	if err != nil {
		f.logger.Printf("failed to remove search from state: %s", err)
		return
	}
}
