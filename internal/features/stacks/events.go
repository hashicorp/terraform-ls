package stacks

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/jobs"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (f *StacksFeature) discover(path string, files []string) error {
	for _, file := range files {
		if ast.IsStacksFilename(file) && !globalAst.IsIgnoredFile(file) {
			f.logger.Printf("discovered module file in %s", path)

			err := f.Store.AddIfNotExists(path)
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
	if languageID != lsp.Stacks.String() {
		return nil, nil
	}

	// Add to state if language ID matches
	if languageID == lsp.Stacks.String() {
		err := f.Store.AddIfNotExists(path)
		if err != nil {
			return ids, err
		}
	}

	// Schedule jobs if state entry exists
	hasStacksRecord := f.Store.Exists(path)
	if !hasStacksRecord {
		return ids, nil
	}

	return f.decodeStacks(ctx, dir, false, true)
}

func (f *StacksFeature) didChange(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	hasModuleRecord := f.Store.Exists(dir.Path())
	if !hasModuleRecord {
		return job.IDs{}, nil
	}

	return f.decodeStacks(ctx, dir, true, true)
}

func (f *StacksFeature) didChangeWatched(ctx context.Context, rawPath string, changeType protocol.FileChangeType, isDir bool) (job.IDs, error) {
	ids := make(job.IDs, 0)

	switch changeType {
	case protocol.Deleted:
		// We don't know whether file or dir is being deleted
		// 1st we just blindly try to look it up as a directory
		hasStackRecord := f.Store.Exists(rawPath)
		if hasStackRecord {
			f.removeIndexedStack(rawPath)
			return ids, nil
		}

		// 2nd we try again assuming it is a file
		parentDir := filepath.Dir(rawPath)
		hasStackRecord = f.Store.Exists(parentDir)
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

		f.decodeStacks(ctx, dir, true, true)

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

		hasModuleRecord := f.Store.Exists(dir.Path())
		if !hasModuleRecord {
			return ids, nil
		}

		f.decodeStacks(ctx, dir, true, true)
	}

	return nil, nil
}

func (f *StacksFeature) decodeStacks(ctx context.Context, dir document.DirHandle, ignoreState bool, isFirstLevel bool) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	parseId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseStackConfiguration(ctx, f.fs, f.Store, path)
		},
		Type: operation.OpTypeParseStacksConfiguration.String(),
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseId)

	// TODO: Implement the following functions where appropriate to stacks
	// Future: LoadModuleMetadata(ctx, f.Store, path)
	// Future: decodeDeclaredModuleCalls(ctx, dir, ignoreState)
	// TODO: PreloadEmbeddedSchema(ctx, f.logger, schemas.FS,
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

	err = f.Store.Remove(rawPath)
	if err != nil {
		f.logger.Printf("failed to remove stack from state: %s", err)
		return
	}
}
