// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/tests/ast"
	"github.com/hashicorp/terraform-ls/internal/features/tests/parser"
	"github.com/hashicorp/terraform-ls/internal/features/tests/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

// ParseTestConfiguration parses the whole test configuration,
// i.e. turns bytes of `*.tftest.hcl` & `*.tfmock.hcl` files into AST ([*hcl.File]).
func ParseTestConfiguration(ctx context.Context, fs ReadOnlyFS, testStore *state.TestStore, testPath string) error {
	record, err := testStore.TestRecordByPath(testPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known
	if record.DiagnosticsState[globalAst.HCLParsingSource] != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(testPath)}
	}

	var files ast.Files
	var diags ast.Diagnostics
	rpcContext := lsctx.DocumentContext(ctx)

	isMatchingLanguageId := (rpcContext.LanguageID == lsp.Test.String() || rpcContext.LanguageID == lsp.Mock.String())

	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if record.DiagnosticsState[globalAst.HCLParsingSource] == operation.OpStateLoaded &&
		rpcContext.IsDidChangeRequest() &&
		isMatchingLanguageId {
		// the file has already been parsed, so only examine this file and not the whole module
		err = testStore.SetDiagnosticsState(testPath, globalAst.HCLParsingSource, operation.OpStateLoading)
		if err != nil {
			return err
		}

		filePath, err := uri.PathFromURI(rpcContext.URI)
		if err != nil {
			return err
		}
		fileName := filepath.Base(filePath)

		pFile, fDiags, err := parser.ParseFile(fs, filePath)
		if err != nil {
			return err
		}
		existingFiles := record.ParsedFiles.Copy()
		existingFiles[ast.FilenameFromName(fileName)] = pFile
		files = existingFiles

		existingDiags, ok := record.Diagnostics[globalAst.HCLParsingSource]
		if !ok {
			existingDiags = make(ast.Diagnostics)
		} else {
			existingDiags = existingDiags.Copy()
		}
		existingDiags[ast.FilenameFromName(fileName)] = fDiags
		diags = existingDiags

	} else {
		// this is the first time file is opened so parse the whole module
		err = testStore.SetDiagnosticsState(testPath, globalAst.HCLParsingSource, operation.OpStateLoading)
		if err != nil {
			return err
		}

		files, diags, err = parser.ParseFiles(fs, testPath)
	}

	sErr := testStore.UpdateParsedFiles(testPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = testStore.UpdateDiagnostics(testPath, globalAst.HCLParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}
