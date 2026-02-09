// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"context"
	"os"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/policy/ast"
	"github.com/hashicorp/terraform-ls/internal/features/policy/jobs"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/protocol"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func (f *PolicyFeature) discover(path string, files []string) error {
	for _, file := range files {
		if ast.IsPolicyFilename(file) && !globalAst.IsIgnoredFile(file) {
			f.logger.Printf("discovered policy file in %s", path)

			err := f.Store.AddIfNotExists(path)
			if err != nil {
				return err
			}

			break
		}
	}

	return nil
}

func (f *PolicyFeature) didOpen(ctx context.Context, dir document.DirHandle, languageID string) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()
	f.logger.Printf("did open %q %q", path, languageID)

	// We need to decide if the path is relevant to us. It can be relevant because
	// a) the walker discovered policy files and created a state entry for them
	// b) the opened file is a policy file
	//
	// Add to state if language ID matches
	if languageID == "terraform-policy" {
		err := f.Store.AddIfNotExists(path)
		if err != nil {
			return ids, err
		}
	}

	// Schedule jobs if state entry exists
	hasPolicyRecord := f.Store.Exists(path)
	if !hasPolicyRecord {
		return ids, nil
	}

	return f.decodePolicy(ctx, dir, false, true)
}

func (f *PolicyFeature) didChange(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	hasPolicyRecord := f.Store.Exists(dir.Path())
	if !hasPolicyRecord {
		return job.IDs{}, nil
	}

	return f.decodePolicy(ctx, dir, true, true)
}

func (f *PolicyFeature) didChangeWatched(ctx context.Context, rawPath string, changeType protocol.FileChangeType, isDir bool) (job.IDs, error) {
	ids := make(job.IDs, 0)

	if changeType == protocol.Deleted {
		// We don't know whether file or dir is being deleted
		// 1st we just blindly try to look it up as a directory
		hasPolicyRecord := f.Store.Exists(rawPath)
		if hasPolicyRecord {
			f.removeIndexedPolicy(rawPath)
			return ids, nil
		}

		// 2nd we try again assuming it is a file
		parentDir := filepath.Dir(rawPath)
		hasPolicyRecord = f.Store.Exists(parentDir)
		if !hasPolicyRecord {
			// Nothing relevant found in the feature state
			return ids, nil
		}

		// and check the parent directory still exists
		fi, err := os.Stat(parentDir)
		if err != nil {
			if os.IsNotExist(err) {
				// if not, we remove the indexed policy
				f.removeIndexedPolicy(rawPath)
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
		// path is a policy path. If so, we need to reparse the policy.
		dir := document.DirHandleFromPath(parentDir)
		hasOpenDocs, err := f.stateStore.DocumentStore.HasOpenDocuments(dir)
		if err != nil {
			f.logger.Printf("error when checking for open documents in path (%q deleted): %s", rawPath, err)
		}
		if !hasOpenDocs {
			return ids, nil
		}

		f.decodePolicy(ctx, dir, true, true)
	}

	if changeType == protocol.Changed || changeType == protocol.Created {
		var dir document.DirHandle
		if isDir {
			dir = document.DirHandleFromPath(rawPath)
		} else {
			docHandle := document.HandleFromPath(rawPath)
			dir = docHandle.Dir
		}

		// Check if the there are open documents for the path and the
		// path is a policy path. If so, we need to reparse the policy.
		hasOpenDocs, err := f.stateStore.DocumentStore.HasOpenDocuments(dir)
		if err != nil {
			f.logger.Printf("error when checking for open documents in path (%q changed): %s", rawPath, err)
		}
		if !hasOpenDocs {
			return ids, nil
		}

		hasPolicyRecord := f.Store.Exists(dir.Path())
		if !hasPolicyRecord {
			return ids, nil
		}

		f.decodePolicy(ctx, dir, true, true)
	}

	return ids, nil
}

func (f *PolicyFeature) removeIndexedPolicy(rawPath string) {
	policyHandle := document.DirHandleFromPath(rawPath)

	err := f.stateStore.JobStore.DequeueJobsForDir(policyHandle)
	if err != nil {
		f.logger.Printf("failed to dequeue jobs for policy: %s", err)
		return
	}

	err = f.Store.Remove(rawPath)
	if err != nil {
		f.logger.Printf("failed to remove policy from state: %s", err)
		return
	}
}

func (f *PolicyFeature) decodePolicy(ctx context.Context, dir document.DirHandle, ignoreState bool, isFirstLevel bool) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	parseId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParsePolicyConfiguration(ctx, f.fs, f.Store, path)
		},
		Type:        op.OpTypeParsePolicyConfiguration.String(),
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
			return jobs.LoadPolicyMetadata(ctx, f.Store, path)
		},
		Type:        op.OpTypeLoadPolicyMetadata.String(),
		DependsOn:   job.IDs{parseId},
		IgnoreState: ignoreState,
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			deferIds := make(job.IDs, 0)
			if jobErr != nil {
				f.logger.Printf("loading policy metadata returned error: %s", jobErr)
			}

			refTargetsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.DecodeReferenceTargets(ctx, f.Store, f.rootFeature, path)
				},
				Type:        op.OpTypeDecodeReferenceTargets.String(),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, refTargetsId)

			refOriginsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.DecodeReferenceOrigins(ctx, f.Store, f.rootFeature, path)
				},
				Type:        op.OpTypeDecodeReferenceOrigins.String(),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, refOriginsId)

			// We don't want to validate nested policy
			if isFirstLevel && validationOptions.EnableEnhancedValidation {
				_, err = f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
					Dir: dir,
					Func: func(ctx context.Context) error {
						return jobs.SchemaPolicyValidation(ctx, f.Store, f.rootFeature, dir.Path())
					},
					Type:        op.OpTypeSchemaPolicyValidation.String(),
					IgnoreState: ignoreState,
				})
				if err != nil {
					return deferIds, err
				}

				_, err = f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
					Dir: dir,
					Func: func(ctx context.Context) error {
						return jobs.ReferenceValidation(ctx, f.Store, f.rootFeature, dir.Path())
					},
					Type:        op.OpTypeReferencePolicyValidation.String(),
					DependsOn:   job.IDs{refOriginsId, refTargetsId},
					IgnoreState: ignoreState,
				})
				if err != nil {
					return deferIds, err
				}
			}

			return deferIds, nil
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, metaId)

	// We don't want to fetch policy data from the registry for nested policy,
	// so we return early.
	if !isFirstLevel {
		return ids, nil
	}

	return ids, nil
}
