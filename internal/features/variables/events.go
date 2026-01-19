// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package variables

import (
	"context"
	"os"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/variables/ast"
	"github.com/hashicorp/terraform-ls/internal/features/variables/jobs"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (f *VariablesFeature) discover(path string, files []string) error {
	for _, file := range files {
		if ast.IsVarsFilename(file) {
			f.logger.Printf("discovered variable file in %s", path)

			err := f.store.AddIfNotExists(path)
			if err != nil {
				return err
			}

			break
		}
	}

	return nil
}

func (f *VariablesFeature) didOpen(ctx context.Context, dir document.DirHandle, languageID string) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	// We need to decide if the path is relevant to us. It can be relevant because
	// a) the walker discovered variable files and created a state entry for them
	// b) the opened file is a variable file
	//
	// Add to state if language ID matches
	if languageID == "terraform-vars" {
		err := f.store.AddIfNotExists(path)
		if err != nil {
			return ids, err
		}
	}

	// Schedule jobs if state entry exists
	hasVariableRecord := f.store.Exists(path)
	if !hasVariableRecord {
		return ids, nil
	}

	return f.decodeVariable(ctx, dir, false)
}

func (f *VariablesFeature) didChange(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	hasVariableRecord := f.store.Exists(dir.Path())
	if !hasVariableRecord {
		return job.IDs{}, nil
	}

	return f.decodeVariable(ctx, dir, true)
}

func (f *VariablesFeature) didChangeWatched(ctx context.Context, rawPath string, changeType protocol.FileChangeType, isDir bool) (job.IDs, error) {
	ids := make(job.IDs, 0)

	if changeType == protocol.Deleted {
		// We don't know whether file or dir is being deleted
		// 1st we just blindly try to look it up as a directory
		hasVariableRecord := f.store.Exists(rawPath)
		if hasVariableRecord {
			f.removeIndexedVariable(rawPath)
			return ids, nil
		}

		// 2nd we try again assuming it is a file
		parentDir := filepath.Dir(rawPath)
		hasVariableRecord = f.store.Exists(parentDir)
		if !hasVariableRecord {
			// Nothing relevant found in the feature state
			return ids, nil
		}

		// and check the parent directory still exists
		fi, err := os.Stat(parentDir)
		if err != nil {
			if os.IsNotExist(err) {
				// if not, we remove the indexed variable
				f.removeIndexedVariable(rawPath)
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
		// path is a variable path. If so, we need to reparse the variable files
		dir := document.DirHandleFromPath(parentDir)
		hasOpenDocs, err := f.stateStore.DocumentStore.HasOpenDocuments(dir)
		if err != nil {
			f.logger.Printf("error when checking for open documents in path (%q deleted): %s", rawPath, err)
		}
		if !hasOpenDocs {
			return ids, nil
		}

		f.decodeVariable(ctx, dir, true)
	}

	if changeType == protocol.Changed {
		docHandle := document.HandleFromPath(rawPath)
		// Check if the there are open documents for the path and the
		// path is a module path. If so, we need to reparse the variable files
		hasOpenDocs, err := f.stateStore.DocumentStore.HasOpenDocuments(docHandle.Dir)
		if err != nil {
			f.logger.Printf("error when checking for open documents in path (%q changed): %s", rawPath, err)
		}
		if !hasOpenDocs {
			return ids, nil
		}

		hasVariableRecord := f.store.Exists(docHandle.Dir.Path())
		if !hasVariableRecord {
			return ids, nil
		}

		f.decodeVariable(ctx, docHandle.Dir, true)
	}

	if changeType == protocol.Created {
		var dir document.DirHandle
		if isDir {
			dir = document.DirHandleFromPath(rawPath)
		} else {
			docHandle := document.HandleFromPath(rawPath)
			dir = docHandle.Dir
		}

		// Check if the there are open documents for the path and the
		// path is a module path. If so, we need to reparse the variable files
		hasOpenDocs, err := f.stateStore.DocumentStore.HasOpenDocuments(dir)
		if err != nil {
			f.logger.Printf("error when checking for open documents in path (%q changed): %s", rawPath, err)
		}
		if !hasOpenDocs {
			return ids, nil
		}

		hasVariableRecord := f.store.Exists(dir.Path())
		if !hasVariableRecord {
			return ids, nil
		}

		f.decodeVariable(ctx, dir, true)
	}

	return ids, nil
}

func (f *VariablesFeature) removeIndexedVariable(rawPath string) {
	modHandle := document.DirHandleFromPath(rawPath)

	err := f.stateStore.JobStore.DequeueJobsForDir(modHandle)
	if err != nil {
		f.logger.Printf("failed to dequeue jobs for variable: %s", err)
		return
	}

	err = f.store.Remove(rawPath)
	if err != nil {
		f.logger.Printf("failed to remove variable from state: %s", err)
		return
	}
}

func (f *VariablesFeature) decodeVariable(ctx context.Context, dir document.DirHandle, ignoreState bool) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	parseVarsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseVariables(ctx, f.fs, f.store, path)
		},
		Type:        op.OpTypeParseVariables.String(),
		IgnoreState: ignoreState,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseVarsId)

	varsRefsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.DecodeVarsReferences(ctx, f.store, f.moduleFeature, path)
		},
		Type:        op.OpTypeDecodeVarsReferences.String(),
		DependsOn:   job.IDs{parseVarsId},
		IgnoreState: ignoreState,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, varsRefsId)

	validationOptions, err := lsctx.ValidationOptions(ctx)
	if err != nil {
		return ids, err
	}
	if validationOptions.EnableEnhancedValidation {
		_, err = f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
			Dir: dir,
			Func: func(ctx context.Context) error {
				return jobs.SchemaVariablesValidation(ctx, f.store, f.moduleFeature, path)
			},
			Type:        op.OpTypeSchemaVarsValidation.String(),
			DependsOn:   job.IDs{parseVarsId},
			IgnoreState: ignoreState,
		})
		if err != nil {
			return ids, err
		}
	}

	return ids, nil
}
