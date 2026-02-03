// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func TestParseModuleConfiguration(t *testing.T) {
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
	testFs := filesystem.NewFilesystem(gs.DocumentStore)

	singleFileModulePath := filepath.Join(testData, "single-file-change-module")

	err = ms.Add(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, testFs, ms, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := ms.ModuleRecordByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// ignore job state
	ctx = job.WithIgnoreState(ctx, true)

	// say we're coming from did_change request
	fooURI, err := filepath.Abs(filepath.Join(singleFileModulePath, "foo.tf"))
	if err != nil {
		t.Fatal(err)
	}
	x := lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Terraform.String(),
		URI:        uri.FromPath(fooURI),
	}
	ctx = lsctx.WithDocumentContext(ctx, x)
	err = ParseModuleConfiguration(ctx, testFs, ms, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	after, err := ms.ModuleRecordByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// test if foo.tf is not the same as first seen
	if before.ParsedModuleFiles["foo.tf"] == after.ParsedModuleFiles["foo.tf"] {
		t.Fatal("file should mismatch")
	}

	// test if main.tf is the same as first seen
	if before.ParsedModuleFiles["main.tf"] != after.ParsedModuleFiles["main.tf"] {
		t.Fatal("file mismatch")
	}

	// examine diags should change for foo.tf
	if before.ModuleDiagnostics[ast.HCLParsingSource]["foo.tf"][0] == after.ModuleDiagnostics[ast.HCLParsingSource]["foo.tf"][0] {
		t.Fatal("diags should mismatch")
	}

	// examine diags should change for main.tf
	if before.ModuleDiagnostics[ast.HCLParsingSource]["main.tf"][0] != after.ModuleDiagnostics[ast.HCLParsingSource]["main.tf"][0] {
		t.Fatal("diags should match")
	}
}

func TestParseModuleConfiguration_ignore_tfvars(t *testing.T) {
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
	testFs := filesystem.NewFilesystem(gs.DocumentStore)

	singleFileModulePath := filepath.Join(testData, "single-file-change-module")

	err = ms.Add(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, testFs, ms, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := ms.ModuleRecordByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// ignore job state
	ctx = job.WithIgnoreState(ctx, true)

	// say we're coming from did_change request
	fooURI, err := filepath.Abs(filepath.Join(singleFileModulePath, "example.tfvars"))
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Tfvars.String(),
		URI:        uri.FromPath(fooURI),
	})
	err = ParseModuleConfiguration(ctx, testFs, ms, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	after, err := ms.ModuleRecordByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// example.tfvars should be missing
	_, beforeExists := before.ParsedModuleFiles["example.tfvars"]
	if beforeExists {
		t.Fatal("example.tfvars file should be missing")
	}
	_, afterExists := after.ParsedModuleFiles["example.tfvars"]
	if afterExists {
		t.Fatal("example.tfvars file should be missing")
	}

	// diags should be missing for example.tfvars
	if _, ok := before.ModuleDiagnostics[ast.HCLParsingSource]["example.tfvars"]; ok {
		t.Fatal("there should be no diags for tfvars files right now")
	}
}
