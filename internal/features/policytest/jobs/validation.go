// Copyright IBM Corp. 2020, 2026
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
	"github.com/hashicorp/terraform-ls/internal/features/policytest/ast"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/policytest/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/decoder/validations"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// SchemaPolicyTestValidation does schema-based validation
// of policytest files (*.policytest.hcl) and produces diagnostics
// associated with any "invalid" parts of code.
//
// It relies on previously parsed AST (via [ParsePolicyTestConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
func SchemaPolicyTestValidation(ctx context.Context, policytestStore *state.PolicyTestStore, rootFeature fdecoder.RootReader, policytestPath string) error {
	policytest, err := policytestStore.PolicyTestRecordByPath(policytestPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if policytest.PolicyTestDiagnosticsState[globalAst.SchemaValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policytestPath)}
	}

	err = policytestStore.SetPolicyTestDiagnosticsState(policytestPath, globalAst.SchemaValidationSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader: policytestStore,
		RootReader:  rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	policytestDecoder, err := d.Path(lang.Path{
		Path:       policytestPath,
		LanguageID: ilsp.PolicyTest.String(),
	})
	if err != nil {
		return err
	}

	var rErr error
	rpcContext := lsctx.DocumentContext(ctx)
	if rpcContext.Method == "textDocument/didChange" && rpcContext.LanguageID == ilsp.PolicyTest.String() {
		filename := path.Base(rpcContext.URI)
		// We only revalidate a single file that changed
		var fileDiags hcl.Diagnostics
		fileDiags, rErr = policytestDecoder.ValidateFile(ctx, filename)

		policytestDiags, ok := policytest.PolicyTestDiagnostics[globalAst.SchemaValidationSource]
		if !ok {
			policytestDiags = make(ast.PolicyTestDiags)
		}
		policytestDiags[ast.PolicyTestFilename(filename)] = fileDiags

		sErr := policytestStore.UpdatePolicyTestDiagnostics(policytestPath, globalAst.SchemaValidationSource, policytestDiags)
		if sErr != nil {
			return sErr
		}
	} else {
		// We validate the whole policytest, e.g. on open
		var diags lang.DiagnosticsMap
		diags, rErr = policytestDecoder.Validate(ctx)

		sErr := policytestStore.UpdatePolicyTestDiagnostics(policytestPath, globalAst.SchemaValidationSource, ast.PolicyTestDiagsFromMap(diags))
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
func ReferenceValidation(ctx context.Context, policytestStore *state.PolicyTestStore, rootFeature fdecoder.RootReader, policytestPath string) error {
	policytest, err := policytestStore.PolicyTestRecordByPath(policytestPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if policytest.PolicyTestDiagnosticsState[globalAst.ReferenceValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policytestPath)}
	}

	err = policytestStore.SetPolicyTestDiagnosticsState(policytestPath, globalAst.ReferenceValidationSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	pathReader := &fdecoder.PathReader{
		StateReader: policytestStore,
		RootReader:  rootFeature,
	}
	pathCtx, err := pathReader.PathContext(lang.Path{
		Path:       policytestPath,
		LanguageID: ilsp.PolicyTest.String(),
	})
	if err != nil {
		return err
	}

	diags := validations.UnreferencedOrigins(ctx, pathCtx)
	return policytestStore.UpdatePolicyTestDiagnostics(policytestPath, globalAst.ReferenceValidationSource, ast.PolicyTestDiagsFromMap(diags))
}
