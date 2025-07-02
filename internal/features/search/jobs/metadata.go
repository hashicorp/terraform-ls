// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/search/ast"
	"github.com/hashicorp/terraform-ls/internal/features/search/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	earlydecoder "github.com/hashicorp/terraform-schema/earlydecoder/search"
)

// LoadSearchMetadata loads data about the search in a version-independent
// way that enables us to decode the rest of the configuration,
// e.g. by knowing provider versions, etc.
func LoadSearchMetadata(ctx context.Context, searchStore *state.SearchStore, searchPath string) error {
	record, err := searchStore.SearchRecordByPath(searchPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if upstream (parsing) job reported no changes

	// Avoid parsing if it is already in progress or already known
	if record.MetaState != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}
	}

	err = searchStore.SetMetaState(searchPath, operation.OpStateLoading)
	if err != nil {
		return err
	}

	meta, diags := earlydecoder.LoadSearch(record.Path(), record.ParsedFiles.AsMap())

	var mErr error
	sErr := searchStore.UpdateMetadata(searchPath, meta, mErr)
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

	sErr = searchStore.UpdateDiagnostics(searchPath, globalAst.HCLParsingSource, existingDiags)
	if sErr != nil {
		return sErr
	}

	return mErr
}
