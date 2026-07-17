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
	"github.com/hashicorp/terraform-ls/internal/features/policy/ast"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/policy/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/policy/decoder/validations"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// SchemaPolicyValidation does schema-based validation
// of policy files (*.policy.hcl) and produces diagnostics
// associated with any "invalid" parts of code.
//
// It relies on previously parsed AST (via [ParsePolicyConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
func SchemaPolicyValidation(ctx context.Context, policyStore *state.PolicyStore, rootFeature fdecoder.RootReader, policyPath string) error {
	policy, err := policyStore.PolicyRecordByPath(policyPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if policy.PolicyDiagnosticsState[globalAst.SchemaValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policyPath)}
	}

	err = policyStore.SetPolicyDiagnosticsState(policyPath, globalAst.SchemaValidationSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader: policyStore,
		RootReader:  rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	policyDecoder, err := d.Path(lang.Path{
		Path:       policyPath,
		LanguageID: ilsp.Policy.String(),
	})
	if err != nil {
		return err
	}

	var rErr error
	rpcContext := lsctx.DocumentContext(ctx)
	if rpcContext.Method == "textDocument/didChange" && rpcContext.LanguageID == ilsp.Policy.String() {
		filename := path.Base(rpcContext.URI)
		// We only revalidate a single file that changed
		var fileDiags hcl.Diagnostics
		fileDiags, rErr = policyDecoder.ValidateFile(ctx, filename)

		policyDiags, ok := policy.PolicyDiagnostics[globalAst.SchemaValidationSource]
		if !ok {
			policyDiags = make(ast.PolicyDiags)
		}
		policyDiags[ast.PolicyFilename(filename)] = fileDiags

		sErr := policyStore.UpdatePolicyDiagnostics(policyPath, globalAst.SchemaValidationSource, policyDiags)
		if sErr != nil {
			return sErr
		}
	} else {
		// We validate the whole policy, e.g. on open
		var diags lang.DiagnosticsMap
		diags, rErr = policyDecoder.Validate(ctx)

		sErr := policyStore.UpdatePolicyDiagnostics(policyPath, globalAst.SchemaValidationSource, ast.PolicyDiagsFromMap(diags))
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
func ReferenceValidation(ctx context.Context, policyStore *state.PolicyStore, rootFeature fdecoder.RootReader, policyPath string) error {
	policy, err := policyStore.PolicyRecordByPath(policyPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if policy.PolicyDiagnosticsState[globalAst.ReferenceValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policyPath)}
	}

	err = policyStore.SetPolicyDiagnosticsState(policyPath, globalAst.ReferenceValidationSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	pathReader := &fdecoder.PathReader{
		StateReader: policyStore,
		RootReader:  rootFeature,
	}
	pathCtx, err := pathReader.PathContext(lang.Path{
		Path:       policyPath,
		LanguageID: ilsp.Policy.String(),
	})
	if err != nil {
		return err
	}

	diags := validations.UnreferencedOrigins(ctx, pathCtx)
	return policyStore.UpdatePolicyDiagnostics(policyPath, globalAst.ReferenceValidationSource, ast.PolicyDiagsFromMap(diags))
}
