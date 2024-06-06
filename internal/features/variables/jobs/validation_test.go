// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/variables/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/uri"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

type ModuleReaderMock struct{}

func (r ModuleReaderMock) ModuleInputs(modPath string) (map[string]tfmod.Variable, error) {
	return map[string]tfmod.Variable{
		"foo": {},
	}, nil
}

func (r ModuleReaderMock) MetadataReady(dir document.DirHandle) (<-chan struct{}, bool, error) {
	return nil, true, nil
}

func TestSchemaVarsValidation_FullModule(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	vs, err := state.NewVariableStore(gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "invalid-tfvars")

	err = vs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didOpen",
		LanguageID: ilsp.Tfvars.String(),
		URI:        "file:///test/terraform.tfvars",
	})
	err = ParseVariables(ctx, fs, vs, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaVariablesValidation(ctx, vs, ModuleReaderMock{}, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := vs.VariableRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 2
	diagsCount := mod.VarsDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}

func TestSchemaVarsValidation_SingleFile(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	vs, err := state.NewVariableStore(gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "invalid-tfvars")

	err = vs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	filePath, err := filepath.Abs(filepath.Join(modPath, "terraform.tfvars"))
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Tfvars.String(),
		URI:        uri.FromPath(filePath),
	})
	err = ParseVariables(ctx, fs, vs, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaVariablesValidation(ctx, vs, ModuleReaderMock{}, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := vs.VariableRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 1
	diagsCount := mod.VarsDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}

func TestSchemaVarsValidation_outsideOfModule(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	vs, err := state.NewVariableStore(gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "standalone-tfvars")

	err = vs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseVariables(ctx, fs, vs, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaVariablesValidation(ctx, vs, ModuleReaderMock{}, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := vs.VariableRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 0
	diagsCount := mod.VarsDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}
