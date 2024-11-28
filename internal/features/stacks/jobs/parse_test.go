// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func TestParseStackConfiguration(t *testing.T) {
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
	testFs := filesystem.NewFilesystem(gs.DocumentStore)

	simpleStackPath := filepath.Join(testData, "simple-stack")

	err = ss.Add(simpleStackPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseStackConfiguration(ctx, testFs, ss, simpleStackPath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := ss.StackRecordByPath(simpleStackPath)
	if err != nil {
		t.Fatal(err)
	}

	// ignore job state
	ctx = job.WithIgnoreState(ctx, true)

	// say we're coming from did_change request
	componentsURI, err := filepath.Abs(filepath.Join(simpleStackPath, "components.tfstack.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	x := lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Stacks.String(),
		URI:        uri.FromPath(componentsURI),
	}
	ctx = lsctx.WithDocumentContext(ctx, x)
	err = ParseStackConfiguration(ctx, testFs, ss, simpleStackPath)
	if err != nil {
		t.Fatal(err)
	}

	after, err := ss.StackRecordByPath(simpleStackPath)
	if err != nil {
		t.Fatal(err)
	}

	componentsFile := ast.StackFilename("components.tfstack.hcl")
	// test if components.tfstack.hcl is not the same as first seen
	if before.ParsedFiles[componentsFile] == after.ParsedFiles[componentsFile] {
		t.Fatal("file should mismatch")
	}

	variablesFile := ast.StackFilename("variables.tfstack.hcl")
	// test if variables.tfstack.hcl is the same as first seen
	if before.ParsedFiles[variablesFile] != after.ParsedFiles[variablesFile] {
		t.Fatal("file mismatch")
	}

	// examine diags should change for components.tfstack.hcl
	if before.Diagnostics[globalAst.HCLParsingSource][componentsFile][0] == after.Diagnostics[globalAst.HCLParsingSource][componentsFile][0] {
		t.Fatal("diags should mismatch")
	}

	// examine diags should not change for variables.tfstack.hcl
	if before.Diagnostics[globalAst.HCLParsingSource][variablesFile][0] != after.Diagnostics[globalAst.HCLParsingSource][variablesFile][0] {
		t.Fatal("diags should match")
	}
}
