// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path"
	"time"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/variables/ast"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/variables/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/variables/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// SchemaVariablesValidation does schema-based validation
// of variable files (*.tfvars) and produces diagnostics
// associated with any "invalid" parts of code.
//
// It relies on previously parsed AST (via [ParseVariables])
// and schema, as provided via [LoadModuleMetadata]).
func SchemaVariablesValidation(ctx context.Context, varStore *state.VariableStore, moduleFeature fdecoder.ModuleReader, modPath string) error {
	mod, err := varStore.VariableRecordByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.VarsDiagnosticsState[globalAst.SchemaValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = varStore.SetVarsDiagnosticsState(modPath, globalAst.SchemaValidationSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	// We only wait a short period for the module to become ready
	// If we have to cancel the validation, we will just run it after the next change
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	wCh, moduleReady, err := moduleFeature.MetadataReady(document.DirHandleFromPath(modPath))
	if err != nil {
		return err
	}
	if !moduleReady {
		select {
		// Wait for module to be ready
		case <-wCh:
		// or for the remaining time to pass
		case <-timer.C:
		// or context cancellation
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader:  varStore,
		ModuleReader: moduleFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	moduleDecoder, err := d.Path(lang.Path{
		Path:       modPath,
		LanguageID: ilsp.Tfvars.String(),
	})
	if err != nil {
		return err
	}

	var rErr error
	rpcContext := lsctx.DocumentContext(ctx)
	if rpcContext.Method == "textDocument/didChange" && rpcContext.LanguageID == ilsp.Tfvars.String() {
		filename := path.Base(rpcContext.URI)
		// We only revalidate a single file that changed
		var fileDiags hcl.Diagnostics
		fileDiags, rErr = moduleDecoder.ValidateFile(ctx, filename)

		varsDiags, ok := mod.VarsDiagnostics[globalAst.SchemaValidationSource]
		if !ok {
			varsDiags = make(ast.VarsDiags)
		}
		varsDiags[ast.VarsFilename(filename)] = fileDiags

		sErr := varStore.UpdateVarsDiagnostics(modPath, globalAst.SchemaValidationSource, varsDiags)
		if sErr != nil {
			return sErr
		}
	} else {
		// We validate the whole module, e.g. on open
		var diags lang.DiagnosticsMap
		diags, rErr = moduleDecoder.Validate(ctx)

		sErr := varStore.UpdateVarsDiagnostics(modPath, globalAst.SchemaValidationSource, ast.VarsDiagsFromMap(diags))
		if sErr != nil {
			return sErr
		}
	}

	return rErr
}
