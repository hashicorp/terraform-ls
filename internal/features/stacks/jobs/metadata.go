// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	earlydecoder "github.com/hashicorp/terraform-schema/earlydecoder/stacks"
)

// LoadStackMetadata loads data about the stack in a version-independent
// way that enables us to decode the rest of the configuration,
// e.g. by knowing provider versions, etc.
func LoadStackMetadata(ctx context.Context, stackStore *state.StackStore, stackPath string) error {
	record, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if upstream (parsing) job reported no changes

	// Avoid parsing if it is already in progress or already known
	if record.MetaState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	err = stackStore.SetMetaState(stackPath, operation.OpStateLoading)
	if err != nil {
		return err
	}

	meta, diags := earlydecoder.LoadStack(record.Path(), record.ParsedFiles.AsMap())

	var mErr error
	sErr := stackStore.UpdateMetadata(stackPath, meta, mErr)
	if sErr != nil {
		return sErr
	}

	if len(diags) <= 0 {
		// no new diagnostics, so return early
		return mErr
	}

	// Merge the new diagnostics with the existing ones
	existingDiags, ok := record.Diagnostics[globalAst.HCLParsingSource]
	if !ok {
		existingDiags = make(ast.Diagnostics)
	} else {
		existingDiags = existingDiags.Copy()
	}

	for fileName, diagnostic := range diags {
		// Convert the filename to an AST filename
		fn := ast.FilenameFromName(fileName)

		// Append the diagnostic to the existing diagnostics if it exists
		existingDiags[fn] = existingDiags[fn].Extend(diagnostic)
	}

	sErr = stackStore.UpdateDiagnostics(stackPath, globalAst.HCLParsingSource, existingDiags)
	if sErr != nil {
		return sErr
	}

	return mErr
}
