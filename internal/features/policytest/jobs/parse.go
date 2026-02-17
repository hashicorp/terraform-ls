// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/ast"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/parser"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

// ParsePolicyTestConfiguration parses the policytest configuration,
// i.e. turns bytes of `*policytest.hcl` files into AST ([*hcl.File]).
func ParsePolicyTestConfiguration(ctx context.Context, fs ReadOnlyFS, policytestStore *state.PolicyTestStore, policytestPath string) error {
	policytest, err := policytestStore.PolicyTestRecordByPath(policytestPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known
	if policytest.PolicyTestDiagnosticsState[globalAst.HCLParsingSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policytestPath)}
	}

	var files ast.PolicyTestFiles
	var diags ast.PolicyTestDiags
	rpcContext := lsctx.DocumentContext(ctx)
	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if policytest.PolicyTestDiagnosticsState[globalAst.HCLParsingSource] == op.OpStateLoaded && rpcContext.IsDidChangeRequest() && rpcContext.LanguageID == ilsp.PolicyTest.String() {
		// the file has already been parsed, so only examine this file and not the whole policytest
		err = policytestStore.SetPolicyTestDiagnosticsState(policytestPath, globalAst.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		filePath, err := uri.PathFromURI(rpcContext.URI)
		if err != nil {
			return err
		}
		fileName := filepath.Base(filePath)

		f, fDiags, err := parser.ParsePolicyTestFile(fs, filePath)
		if err != nil {
			return err
		}
		existingFiles := policytest.ParsedPolicyTestFiles.Copy()
		existingFiles[ast.PolicyTestFilename(fileName)] = f
		files = existingFiles

		existingDiags, ok := policytest.PolicyTestDiagnostics[globalAst.HCLParsingSource]
		if !ok {
			existingDiags = make(ast.PolicyTestDiags)
		} else {
			existingDiags = existingDiags.Copy()
		}
		existingDiags[ast.PolicyTestFilename(fileName)] = fDiags
		diags = existingDiags
	} else {
		// this is the first time file is opened so parse the whole policytest
		err = policytestStore.SetPolicyTestDiagnosticsState(policytestPath, globalAst.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		files, diags, err = parser.ParsePolicyTestFiles(fs, policytestPath)
	}

	if err != nil {
		return err
	}

	sErr := policytestStore.UpdateParsedPolicyTestFiles(policytestPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = policytestStore.UpdatePolicyTestDiagnostics(policytestPath, globalAst.HCLParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}
