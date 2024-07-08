// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// TODO: find a place to register a watcher for this .terraform-version file (put this in a follow-up issue), for now reloading the window is okay enough
// https://github.com/hashicorp/terraform-ls/blob/c47b2e51a0ca08628a596eca8d5cd11d005b7d87/internal/langserver/handlers/initialized.go#L25

// LoadTerraformVersion loads the terraform version from the .terraform-version
// file in the stack directory.
func LoadTerraformVersion(ctx context.Context, fs ReadOnlyFS, stackStore *state.StackStore, stackPath string) error {
	stackRecord, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// Avoid parsing if it is already in progress or already known
	if stackRecord.RequiredTerraformVersionState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	err = stackStore.SetTerraformVersionState(stackPath, operation.OpStateLoading)
	if err != nil {
		return err
	}

	// read version file
	v, err := fs.ReadFile(filepath.Join(stackPath, ".terraform-version"))
	if err != nil {
		updateErr := stackStore.SetTerraformVersionError(stackPath, err)
		if updateErr != nil {
			return updateErr
		}

		return err
	}

	// parse version
	version, err := version.NewVersion(strings.TrimSpace(string(v)))
	if err != nil {
		updateErr := stackStore.SetTerraformVersionError(stackPath, err)
		if updateErr != nil {
			return updateErr
		}

		return err
	}

	return stackStore.SetTerraformVersion(stackPath, version)
}
