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

// ParseModuleConfiguration parses the Stack configuration,
// i.e. turns bytes of `*.tfstack.hcl` files into AST ([*hcl.File]).
func ParseStackConfiguration(ctx context.Context, fs ReadOnlyFS, stackStore *state.StackStore, stackPath string) error {
	record, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known
	if record.StackDiagnosticsState[globalAst.HCLParsingSource] != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	var files ast.StackFiles
	var diags ast.StackDiags
	rpcContext := lsctx.DocumentContext(ctx)
	
	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if record.StackDiagnosticsState[globalAst.HCLParsingSource] == operation.OpStateLoaded &&
		rpcContext.IsDidChangeRequest() &&
		rpcContext.LanguageID == lsp.Stacks.String() {
		// the file has already been parsed, so only examine this file and not the whole module
		err = stackStore.SetStackDiagnosticsState(stackPath, globalAst.HCLParsingSource, operation.OpStateLoading)
		if err != nil {
			return err
		}

		filePath, err := uri.PathFromURI(rpcContext.URI)
		if err != nil {
			return err
		}
		fileName := filepath.Base(filePath)

		stackFile, stackFileDiags, err := parser.ParseStackFile(fs, filePath)
		if err != nil {
			return err
		}
		existingFiles := record.ParsedStackFiles.Copy()
		existingFiles[ast.StackFilename(fileName)] = stackFile
		files = existingFiles

		existingDiags, ok := record.StackDiagnostics[globalAst.HCLParsingSource]
		if !ok {
			existingDiags = make(ast.StackDiags)
		} else {
			existingDiags = existingDiags.Copy()
		}
		existingDiags[ast.StackFilename(fileName)] = stackFileDiags
		diags = existingDiags

		sErr := stackStore.UpdateParsedStackFiles(stackPath, files, err)
		if sErr != nil {
			return sErr
		}
	
		sErr = stackStore.UpdateStackDiagnostics(stackPath, globalAst.HCLParsingSource, diags)
		if sErr != nil {
			return sErr
		}

	} else {
		// this is the first time file is opened so parse the whole module
		err = stackStore.SetStackDiagnosticsState(stackPath, globalAst.HCLParsingSource, operation.OpStateLoading)
		if err != nil {
			return err
		}
		files, diags, err = parser.ParseStackFiles(fs, stackPath)

		sErr := stackStore.UpdateParsedStackFiles(stackPath, files, err)
		if sErr != nil {
			return sErr
		}
	
		sErr = stackStore.UpdateStackDiagnostics(stackPath, globalAst.HCLParsingSource, diags)
		if sErr != nil {
			return sErr
		}
	}

	if err != nil {
		return err
	}

	return err
}

func ParseDeployConfiguration(ctx context.Context, fs ReadOnlyFS, stackStore *state.StackStore, stackPath string) error {
	record, err := stackStore.StackRecordByPath(stackPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known
	if record.DeployDiagnosticsState[globalAst.HCLParsingSource] != operation.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}
	}

	var files ast.DeployFiles
	var diags ast.DeployDiags
	rpcContext := lsctx.DocumentContext(ctx)
	
	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if record.DeployDiagnosticsState[globalAst.HCLParsingSource] == operation.OpStateLoaded &&
		rpcContext.IsDidChangeRequest() &&
		rpcContext.LanguageID == lsp.Deploy.String() {
		// the file has already been parsed, so only examine this file and not the whole module
		err = stackStore.SetDeployDiagnosticsState(stackPath, globalAst.HCLParsingSource, operation.OpStateLoading)
		if err != nil {
			return err
		}

		filePath, err := uri.PathFromURI(rpcContext.URI)
		if err != nil {
			return err
		}
		fileName := filepath.Base(filePath)

		deployFile, deployFileDiags, err := parser.ParseDeployFile(fs, filePath)
		if err != nil {
			return err
		}
		existingFiles := record.ParsedDeployFiles.Copy()
		existingFiles[ast.DeployFilename(fileName)] = deployFile
		files = existingFiles

		existingDiags, ok := record.DeployDiagnostics[globalAst.HCLParsingSource]
		if !ok {
			existingDiags = make(ast.DeployDiags)
		} else {
			existingDiags = existingDiags.Copy()
		}
		existingDiags[ast.DeployFilename(fileName)] = deployFileDiags
		diags = existingDiags

	} else {
		// this is the first time file is opened so parse the whole module
		err = stackStore.SetDeployDiagnosticsState(stackPath, globalAst.HCLParsingSource, operation.OpStateLoading)
		if err != nil {
			return err
		}
		files, diags, err = parser.ParseDeployFiles(fs, stackPath)
	}

	if err != nil {
		return err
	}

	sErr := stackStore.UpdateParsedDeployFiles(stackPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = stackStore.UpdateDeployDiagnostics(stackPath, globalAst.HCLParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}
