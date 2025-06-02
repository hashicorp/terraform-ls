// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/parser"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

// ParseStackConfiguration parses the whole Stack configuration,
// i.e. turns bytes of `*.tfcomponent.hcl`, `*.tfstack.hcl` & `*.tfdeploy.hcl` files into AST ([*hcl.File]).
func ParseStackConfiguration(ctx context.Context, fs ReadOnlyFS, stackStore *state.StackStore, stackPath string) error {
	record, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known
	if record.DiagnosticsState[globalAst.HCLParsingSource] != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	var files ast.Files
	var diags ast.Diagnostics
	rpcContext := lsctx.DocumentContext(ctx)

	isMatchingLanguageId := (rpcContext.LanguageID == lsp.Stacks.String() || rpcContext.LanguageID == lsp.Deploy.String())

	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if record.DiagnosticsState[globalAst.HCLParsingSource] == operation.OpStateLoaded &&
		rpcContext.IsDidChangeRequest() &&
		isMatchingLanguageId {
		// the file has already been parsed, so only examine this file and not the whole module
		err = stackStore.SetDiagnosticsState(stackPath, globalAst.HCLParsingSource, operation.OpStateLoading)
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
		err = stackStore.SetDiagnosticsState(stackPath, globalAst.HCLParsingSource, operation.OpStateLoading)
		if err != nil {
			return err
		}

		files, diags, err = parser.ParseFiles(fs, stackPath)
	}

	sErr := stackStore.UpdateParsedFiles(stackPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = stackStore.UpdateDiagnostics(stackPath, globalAst.HCLParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}
