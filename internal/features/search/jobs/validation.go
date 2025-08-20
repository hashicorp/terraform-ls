// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/search/ast"
	searchDecoder "github.com/hashicorp/terraform-ls/internal/features/search/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/search/decoder/validations"
	"github.com/hashicorp/terraform-ls/internal/features/search/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

func SchemaSearchValidation(ctx context.Context, searchStore *state.SearchStore, moduleFeature searchDecoder.ModuleReader, rootFeature searchDecoder.RootReader, searchPath string) error {
	rpcContext := lsctx.DocumentContext(ctx)

	record, err := searchStore.GetSearchRecordByPath(searchPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if record.DiagnosticsState[globalAst.SchemaValidationSource] != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}
	}

	err = searchStore.SetDiagnosticsState(searchPath, globalAst.SchemaValidationSource, operation.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&searchDecoder.PathReader{
		StateReader:  searchStore,
		ModuleReader: moduleFeature,
		RootReader:   rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	var rErr error
	if rpcContext.Method == "textDocument/didChange" {
		// We validate only the file that has changed
		// This means only creating a decoder for the file type that has changed
		decoder, err := d.Path(lang.Path{
			Path:       searchPath,
			LanguageID: rpcContext.LanguageID,
		})
		if err != nil {
			return err
		}

		filename := path.Base(rpcContext.URI)

		var fileDiags hcl.Diagnostics
		fileDiags, rErr = decoder.ValidateFile(ctx, filename)

		diags, ok := record.Diagnostics[globalAst.SchemaValidationSource]
		if !ok {
			diags = make(ast.Diagnostics)
		}
		diags[ast.FilenameFromName(filename)] = fileDiags

		sErr := searchStore.UpdateDiagnostics(searchPath, globalAst.SchemaValidationSource, diags)
		if sErr != nil {
			return sErr
		}
	} else {
		// We validate the whole search, and so need to create decoders for
		// all the file types in the search
		searchDecoder, err := d.Path(lang.Path{
			Path:       searchPath,
			LanguageID: ilsp.Search.String(),
		})
		if err != nil {
			return err
		}

		diags := make(lang.DiagnosticsMap)

		searchDiags, err := searchDecoder.Validate(ctx)
		if err != nil {
			// TODO: Should we really return here or continue with the other decoders?
			// Is this really a complete fail case? Shouldn't a failure in a search file
			// not prevent the deploy file from being validated?
			return err
		}
		diags = diags.Extend(searchDiags)

		sErr := searchStore.UpdateDiagnostics(searchPath, globalAst.SchemaValidationSource, ast.DiagnosticsFromMap(diags))
		if sErr != nil {
			return sErr
		}
	}

	return rErr
}

// ReferenceValidation does validation based on (mis)matched
// reference origins and targets, to flag up "orphaned" references.
//
// It relies on [DecodeReferenceTargets] and [DecodeReferenceOrigins]
// to supply both origins and targets to compare.
func ReferenceValidation(ctx context.Context, searchStore *state.SearchStore, moduleFeature searchDecoder.ModuleReader, rootFeature searchDecoder.RootReader, searchPath string) error {
	record, err := searchStore.GetSearchRecordByPath(searchPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if record.DiagnosticsState[globalAst.ReferenceValidationSource] != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}
	}

	err = searchStore.SetDiagnosticsState(searchPath, globalAst.ReferenceValidationSource, operation.OpStateLoading)
	if err != nil {
		return err
	}

	pathReader := &searchDecoder.PathReader{
		StateReader:  searchStore,
		ModuleReader: moduleFeature,
		RootReader:   rootFeature,
	}

	searchDecoder, err := pathReader.PathContext(lang.Path{
		Path:       searchPath,
		LanguageID: ilsp.Search.String(),
	})
	if err != nil {
		return err
	}

	diags := validations.UnreferencedOrigins(ctx, searchDecoder)

	return searchStore.UpdateDiagnostics(searchPath, globalAst.ReferenceValidationSource, ast.DiagnosticsFromMap(diags))
}
