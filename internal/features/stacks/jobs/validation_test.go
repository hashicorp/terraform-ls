// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

type ModuleReaderMock struct{}

func (m ModuleReaderMock) LocalModuleMeta(modulePath string) (*tfmod.Meta, error) {
	return nil, nil
}

type RootReaderMock struct{}

func (r RootReaderMock) InstalledModuleCalls(modPath string) (map[string]tfmod.InstalledModuleCall, error) {
	return nil, nil
}

func (r RootReaderMock) TerraformVersion(modPath string) *version.Version {
	return nil
}

func (r RootReaderMock) InstalledModulePath(rootPath string, normalizedSource string) (string, bool) {
	return "", false
}

func TestSchemaStackValidation_FullStack(t *testing.T) {
	runTestSchemaStackValidation_FullStack(t, struct {
		folderName string
		extension  string
	}{folderName: "invalid-stack", extension: "tfcomponent.hcl"})
	runTestSchemaStackValidation_FullStack(t, struct {
		folderName string
		extension  string
	}{folderName: "invalid-stack-legacy-extension", extension: "tfstack.hcl"})
}

func runTestSchemaStackValidation_FullStack(t *testing.T, tc struct {
	folderName string
	extension  string
}) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewStackStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	stackPath := filepath.Join(testData, tc.folderName)

	err = ms.Add(stackPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didOpen",
		LanguageID: ilsp.Stacks.String(),
		URI:        "file:///test/variables." + tc.extension,
	})
	err = ParseStackConfiguration(ctx, fs, ms, stackPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaStackValidation(ctx, ms, ModuleReaderMock{}, RootReaderMock{}, stackPath)
	if err != nil {
		t.Fatal(err)
	}

	record, err := ms.StackRecordByPath(stackPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 3
	diagsCount := record.Diagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}

func TestSchemaStackValidation_SingleFile(t *testing.T) {
	runTestSchemaStackValidation_SingleFile(t, struct {
		folderName string
		extension  string
	}{folderName: "invalid-stack", extension: "tfcomponent.hcl"})
	runTestSchemaStackValidation_SingleFile(t, struct {
		folderName string
		extension  string
	}{folderName: "invalid-stack-legacy-extension", extension: "tfstack.hcl"})
}

func runTestSchemaStackValidation_SingleFile(t *testing.T, tc struct {
	folderName string
	extension  string
}) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss, err := state.NewStackStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	stackPath := filepath.Join(testData, tc.folderName)

	err = ss.Add(stackPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Stacks.String(),
		URI:        "file:///test/variables." + tc.extension,
	})
	err = ParseStackConfiguration(ctx, fs, ss, stackPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaStackValidation(ctx, ss, ModuleReaderMock{}, RootReaderMock{}, stackPath)
	if err != nil {
		t.Fatal(err)
	}

	record, err := ss.StackRecordByPath(stackPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 2
	diagsCount := record.Diagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}
