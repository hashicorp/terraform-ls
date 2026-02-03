// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/variables/ast"
	"github.com/hashicorp/terraform-ls/internal/features/variables/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func TestParseVariables(t *testing.T) {
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
	testFs := filesystem.NewFilesystem(gs.DocumentStore)

	singleFileModulePath := filepath.Join(testData, "single-file-change-module")

	err = vs.Add(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseVariables(ctx, testFs, vs, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := vs.VariableRecordByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// ignore job state
	ctx = job.WithIgnoreState(ctx, true)

	// say we're coming from did_change request
	filePath, err := filepath.Abs(filepath.Join(singleFileModulePath, "example.tfvars"))
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Tfvars.String(),
		URI:        uri.FromPath(filePath),
	})
	err = ParseVariables(ctx, testFs, vs, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	after, err := vs.VariableRecordByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// example.tfvars should not be the same as first seen
	if before.ParsedVarsFiles["example.tfvars"] == after.ParsedVarsFiles["example.tfvars"] {
		t.Fatal("file should mismatch")
	}

	beforeDiags := before.VarsDiagnostics[globalAst.HCLParsingSource]
	afterDiags := after.VarsDiagnostics[globalAst.HCLParsingSource]

	// diags should change for example.tfvars
	if beforeDiags[ast.VarsFilename("example.tfvars")][0] == afterDiags[ast.VarsFilename("example.tfvars")][0] {
		t.Fatal("diags should mismatch")
	}

	if before.ParsedVarsFiles["nochange.tfvars"] != after.ParsedVarsFiles["nochange.tfvars"] {
		t.Fatal("unchanged file should match")
	}

	if beforeDiags[ast.VarsFilename("nochange.tfvars")][0] != afterDiags[ast.VarsFilename("nochange.tfvars")][0] {
		t.Fatal("diags should match for unchanged file")
	}
}
