// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"context"
	"os"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"github.com/hashicorp/terraform-ls/internal/features/tests/ast"
	"github.com/hashicorp/terraform-ls/internal/features/tests/jobs"
	"github.com/hashicorp/terraform-ls/internal/features/tests/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (f *TestsFeature) discover(path string, files []string) error {
	for _, file := range files {
		if globalAst.IsIgnoredFile(file) {
			continue
		}

		if ast.IsTestFilename(file) || ast.IsMockFilename(file) {
			f.logger.Printf("discovered test file in %s", path)

			err := f.store.AddIfNotExists(path)
			if err != nil {
				return err
			}

			break
		}
	}

	return nil
}

func (f *TestsFeature) didOpen(ctx context.Context, dir document.DirHandle, languageID string) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()
	f.logger.Printf("did open %q %q", path, languageID)

	// We need to decide if the path is relevant to us
	if languageID != lsp.Test.String() && languageID != lsp.Mock.String() {
		return ids, nil
	}

	// Add to state as path is relevant
	err := f.store.AddIfNotExists(path)
	if err != nil {
		return ids, err
	}

	decodeIds, err := f.decodeTest(ctx, dir, false, true)
	if err != nil {
		return ids, err
	}
	ids = append(ids, decodeIds...)

	return ids, err
}

func (f *TestsFeature) didChange(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	hasTestRecord := f.store.Exists(dir.Path())
	if !hasTestRecord {
		return job.IDs{}, nil
	}

	return f.decodeTest(ctx, dir, true, true)
}

func (f *TestsFeature) didChangeWatched(ctx context.Context, rawPath string, changeType protocol.FileChangeType, isDir bool) (job.IDs, error) {
	ids := make(job.IDs, 0)

	switch changeType {
	case protocol.Deleted:
		// We don't know whether file or dir is being deleted
		// 1st we just blindly try to look it up as a directory
		hasTestRecord := f.store.Exists(rawPath)
		if hasTestRecord {
			f.removeIndexedTest(rawPath)
			return ids, nil
		}

		// 2nd we try again assuming it is a file
		parentDir := filepath.Dir(rawPath)
		hasTestRecord = f.store.Exists(parentDir)
		if !hasTestRecord {
			// Nothing relevant found in the feature state
			return ids, nil
		}

		// and check the parent directory still exists
		fi, err := os.Stat(parentDir)
		if err != nil {
			if os.IsNotExist(err) {
				// if not, we remove the indexed module
				f.removeIndexedTest(rawPath)
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

		return f.decodeTest(ctx, dir, true, true)

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

		return f.decodeTest(ctx, dir, true, true)
	}

	return nil, nil
}

func (f *TestsFeature) decodeTest(ctx context.Context, dir document.DirHandle, ignoreState bool, isFirstLevel bool) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	parseId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseTestConfiguration(ctx, f.fs, f.store, path)
		},
		Type:        op.OpTypeParseTestConfiguration.String(),
		IgnoreState: ignoreState,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseId)

	validationOptions, _ := lsctx.ValidationOptions(ctx)

	metaId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.LoadTestMetadata(ctx, f.store, path)
		},
		Type:        op.OpTypeLoadTestMetadata.String(),
		DependsOn:   job.IDs{parseId},
		IgnoreState: ignoreState,
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			deferIds := make(job.IDs, 0)
			if jobErr != nil {
				f.logger.Printf("loading module metadata returned error: %s", jobErr)
			}

			spawnedIds, err := loadTestModuleSources(ctx, f.store, f.bus, path)
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, spawnedIds...)

			refTargetsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.DecodeReferenceTargets(ctx, f.store, path, f.moduleFeature, f.rootFeature)
				},
				Type:        op.OpTypeDecodeTestReferenceTargets.String(),
				DependsOn:   spawnedIds,
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, refTargetsId)

			refOriginsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.DecodeReferenceOrigins(ctx, f.store, path, f.moduleFeature, f.rootFeature)
				},
				Type:        op.OpTypeDecodeTestReferenceOrigins.String(),
				DependsOn:   spawnedIds,
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, refOriginsId)

			if validationOptions.EnableEnhancedValidation {
				_, err = f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
					Dir: dir,
					Func: func(ctx context.Context) error {
						return jobs.SchemaTestValidation(ctx, f.store, dir.Path(), f.moduleFeature, f.rootFeature)
					},
					Type:        op.OpTypeSchemaTestValidation.String(),
					DependsOn:   spawnedIds,
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

func (f *TestsFeature) removeIndexedTest(rawPath string) {
	testHandle := document.DirHandleFromPath(rawPath)

	err := f.stateStore.JobStore.DequeueJobsForDir(testHandle)
	if err != nil {
		f.logger.Printf("failed to dequeue jobs for test: %s", err)
		return
	}

	err = f.store.Remove(rawPath)
	if err != nil {
		f.logger.Printf("failed to remove test from state: %s", err)
		return
	}
}

func loadTestModuleSources(ctx context.Context, testStore *state.TestStore, bus *eventbus.EventBus, testPath string) (job.IDs, error) {
	ids := make(job.IDs, 0)

	_, err := testStore.TestRecordByPath(testPath)
	if err != nil {
		return ids, err
	}

	// TODO! load the adjacent Terraform module (usually ../)
	// TODO load the run -> module block sources
	// TODO load the mock_provider block sources

	return ids, nil
}
