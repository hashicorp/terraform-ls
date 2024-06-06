// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/modules/ast"
	"github.com/hashicorp/terraform-ls/internal/features/modules/parser"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

// ParseModuleConfiguration parses the module configuration,
// i.e. turns bytes of `*.tf` files into AST ([*hcl.File]).
func ParseModuleConfiguration(ctx context.Context, fs ReadOnlyFS, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known
	if mod.ModuleDiagnosticsState[globalAst.HCLParsingSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	var files ast.ModFiles
	var diags ast.ModDiags
	rpcContext := lsctx.DocumentContext(ctx)
	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if mod.ModuleDiagnosticsState[globalAst.HCLParsingSource] == op.OpStateLoaded && rpcContext.IsDidChangeRequest() && rpcContext.LanguageID == ilsp.Terraform.String() {
		// the file has already been parsed, so only examine this file and not the whole module
		err = modStore.SetModuleDiagnosticsState(modPath, globalAst.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		filePath, err := uri.PathFromURI(rpcContext.URI)
		if err != nil {
			return err
		}
		fileName := filepath.Base(filePath)

		f, fDiags, err := parser.ParseModuleFile(fs, filePath)
		if err != nil {
			return err
		}
		existingFiles := mod.ParsedModuleFiles.Copy()
		existingFiles[ast.ModFilename(fileName)] = f
		files = existingFiles

		existingDiags, ok := mod.ModuleDiagnostics[globalAst.HCLParsingSource]
		if !ok {
			existingDiags = make(ast.ModDiags)
		} else {
			existingDiags = existingDiags.Copy()
		}
		existingDiags[ast.ModFilename(fileName)] = fDiags
		diags = existingDiags
	} else {
		// this is the first time file is opened so parse the whole module
		err = modStore.SetModuleDiagnosticsState(modPath, globalAst.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		files, diags, err = parser.ParseModuleFiles(fs, modPath)
	}

	if err != nil {
		return err
	}

	sErr := modStore.UpdateParsedModuleFiles(modPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = modStore.UpdateModuleDiagnostics(modPath, globalAst.HCLParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}
