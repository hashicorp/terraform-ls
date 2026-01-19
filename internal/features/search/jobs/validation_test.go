// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/search/state"
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

func TestSchemaSearchValidation_FullSearch(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	searchPath := filepath.Join(testData, "invalid-search")

	err = ms.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didOpen",
		LanguageID: ilsp.Search.String(),
		URI:        "file:///test/variables.tfquery.hcl",
	})
	err = ParseSearchConfiguration(ctx, fs, ms, searchPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaSearchValidation(ctx, ms, ModuleReaderMock{}, RootReaderMock{}, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	record, err := ms.GetSearchRecordByPath(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 2
	diagsCount := record.Diagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}

func TestSchemaSearchValidation_SingleFile(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	searchPath := filepath.Join(testData, "invalid-search")

	err = ss.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Search.String(),
		URI:        "file:///test/config.tfquery.hcl",
	})
	err = ParseSearchConfiguration(ctx, fs, ss, searchPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaSearchValidation(ctx, ss, ModuleReaderMock{}, RootReaderMock{}, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	record, err := ss.GetSearchRecordByPath(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 1
	diagsCount := record.Diagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}
