// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	idecoder "github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/tests/ast"
	fdecoder "github.com/hashicorp/terraform-ls/internal/features/tests/decoder"
	"github.com/hashicorp/terraform-ls/internal/features/tests/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// SchemaTestValidation does schema-based validation
// of test files (*.tftest.hcl), mock files (*.tfmock.hcl)
// and produces diagnostics associated with any "invalid" parts of code.
//
// It relies on previously parsed AST (via [ParseModuleConfiguration]),
// core schema of appropriate version (as obtained via [GetTerraformVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
func SchemaTestValidation(ctx context.Context, testStore *state.TestStore, testPath string, moduleFeature fdecoder.ModuleReader, rootFeature fdecoder.RootReader) error {
	mod, err := testStore.TestRecordByPath(testPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.DiagnosticsState[globalAst.SchemaValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(testPath)}
	}

	err = testStore.SetDiagnosticsState(testPath, globalAst.SchemaValidationSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader:  testStore,
		ModuleReader: moduleFeature,
		RootReader:   rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	diags := make(lang.DiagnosticsMap)

	testDecoder, err := d.Path(lang.Path{
		Path:       testPath,
		LanguageID: ilsp.Test.String(),
	})
	if err != nil {
		return err
	}
	testDiags, err := testDecoder.Validate(ctx)
	if err != nil {
		return err
	}
	diags = diags.Extend(testDiags)

	mockDecoder, err := d.Path(lang.Path{
		Path:       testPath,
		LanguageID: ilsp.Mock.String(),
	})
	if err != nil {
		return err
	}
	mockDiags, err := mockDecoder.Validate(ctx)
	if err != nil {
		return err
	}
	diags = diags.Extend(mockDiags)

	return testStore.UpdateDiagnostics(testPath, globalAst.SchemaValidationSource, ast.DiagnosticsFromMap(diags))
}
