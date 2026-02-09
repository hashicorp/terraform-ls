// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/policy/ast"
	"github.com/hashicorp/terraform-ls/internal/features/policy/parser"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

// ParsePolicyConfiguration parses the policy configuration,
// i.e. turns bytes of `*policy.hcl` files into AST ([*hcl.File]).
func ParsePolicyConfiguration(ctx context.Context, fs ReadOnlyFS, policyStore *state.PolicyStore, policyPath string) error {
	policy, err := policyStore.PolicyRecordByPath(policyPath)
	if err != nil {
		return err
	}

	// TODO: Avoid parsing if the content matches existing AST

	// Avoid parsing if it is already in progress or already known
	if policy.PolicyDiagnosticsState[globalAst.HCLParsingSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(policyPath)}
	}

	var files ast.PolicyFiles
	var diags ast.PolicyDiags
	rpcContext := lsctx.DocumentContext(ctx)
	// Only parse the file that's being changed/opened, unless this is 1st-time parsing
	if policy.PolicyDiagnosticsState[globalAst.HCLParsingSource] == op.OpStateLoaded && rpcContext.IsDidChangeRequest() && rpcContext.LanguageID == ilsp.Policy.String() {
		// the file has already been parsed, so only examine this file and not the whole policy
		err = policyStore.SetPolicyDiagnosticsState(policyPath, globalAst.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		filePath, err := uri.PathFromURI(rpcContext.URI)
		if err != nil {
			return err
		}
		fileName := filepath.Base(filePath)

		f, fDiags, err := parser.ParsePolicyFile(fs, filePath)
		if err != nil {
			return err
		}
		existingFiles := policy.ParsedPolicyFiles.Copy()
		existingFiles[ast.PolicyFilename(fileName)] = f
		files = existingFiles

		existingDiags, ok := policy.PolicyDiagnostics[globalAst.HCLParsingSource]
		if !ok {
			existingDiags = make(ast.PolicyDiags)
		} else {
			existingDiags = existingDiags.Copy()
		}
		existingDiags[ast.PolicyFilename(fileName)] = fDiags
		diags = existingDiags
	} else {
		// this is the first time file is opened so parse the whole policy
		err = policyStore.SetPolicyDiagnosticsState(policyPath, globalAst.HCLParsingSource, op.OpStateLoading)
		if err != nil {
			return err
		}

		files, diags, err = parser.ParsePolicyFiles(fs, policyPath)
	}

	if err != nil {
		return err
	}

	sErr := policyStore.UpdateParsedPolicyFiles(policyPath, files, err)
	if sErr != nil {
		return sErr
	}

	sErr = policyStore.UpdatePolicyDiagnostics(policyPath, globalAst.HCLParsingSource, diags)
	if sErr != nil {
		return sErr
	}

	return err
}
