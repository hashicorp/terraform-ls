// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

type RootReaderMock struct{}

func (r RootReaderMock) InstalledModuleCalls(modPath string) (map[string]tfmod.InstalledModuleCall, error) {
	return nil, nil
}

func (r RootReaderMock) TerraformVersion(modPath string) *version.Version {
	return nil
}

func TestSchemaModuleValidation_FullModule(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "invalid-config")

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didOpen",
		LanguageID: ilsp.Terraform.String(),
		URI:        "file:///test/variables.tf",
	})
	err = ParseModuleConfiguration(ctx, fs, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaModuleValidation(ctx, ms, RootReaderMock{}, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := ms.ModuleRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 5
	diagsCount := mod.ModuleDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}

func TestSchemaModuleValidation_SingleFile(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "invalid-config")

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Terraform.String(),
		URI:        "file:///test/variables.tf",
	})
	err = ParseModuleConfiguration(ctx, fs, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaModuleValidation(ctx, ms, RootReaderMock{}, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := ms.ModuleRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 3
	diagsCount := mod.ModuleDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}
