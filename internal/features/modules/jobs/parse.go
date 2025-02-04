// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"log"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
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
	log.Printf("ParseModuleConfiguration called for path: %s", modPath)

	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	rpcContext := lsctx.DocumentContext(ctx)
	fileName := filepath.Base(rpcContext.URI)

	log.Printf("Current module state: %v, isChange: %v, ignoreState: %v, fileName: %s",
		mod.ModuleDiagnosticsState[globalAst.HCLParsingSource],
		rpcContext.IsDidChangeRequest(),
		job.IgnoreState(ctx),
		fileName)

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known AND we've already parsed this file
	// if mod.ModuleDiagnosticsState[globalAst.HCLParsingSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
	// 	log.Printf("Early return from ParseModuleConfiguration. Module state: %+v", mod)
	// 	log.Printf("module diagnostics state: %+v", mod.ModuleDiagnosticsState[globalAst.HCLParsingSource])
	// 	return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	// }

	var files ast.ModFiles
	var diags ast.ModDiags
	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if mod.ModuleDiagnosticsState[globalAst.HCLParsingSource] == op.OpStateLoaded && rpcContext.IsDidChangeRequest() && rpcContext.LanguageID == ilsp.Terraform.String() {
		log.Printf("Parsing single file due to change: %s", rpcContext.URI)

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
		log.Printf("Parsing entire module at: %s", modPath)

		// this is the first time file is opened so parse the whole module
		err = modStore.SetModuleDiagnosticsState(modPath, globalAst.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		files, diags, err = parser.ParseModuleFiles(fs, modPath)
	}
	fileNames := make([]string, 0, len(files))
	for name := range files {
		fileNames = append(fileNames, string(name))
	}
	log.Printf("Parsed files: %v", fileNames)

	if err != nil {
		return err
	}

	sErr := modStore.UpdateParsedModuleFiles(modPath, files, err)
	if sErr != nil {
		return sErr
	}

	log.Printf("Updating module diagnostics. Current diagnostics: %+v", diags)
	sErr = modStore.UpdateModuleDiagnostics(modPath, globalAst.HCLParsingSource, diags)
	if sErr != nil {
		return sErr
	}
	log.Printf("After updating diagnostics. Module state: %+v", mod.ModuleDiagnostics)

	return err
}
