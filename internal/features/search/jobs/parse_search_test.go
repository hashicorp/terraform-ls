// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/search/ast"
	"github.com/hashicorp/terraform-ls/internal/features/search/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func TestParseSearchConfiguration(t *testing.T) {
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
	testFs := filesystem.NewFilesystem(gs.DocumentStore)

	simpleSearchPath := filepath.Join(testData, "simple-search")

	err = ss.Add(simpleSearchPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseSearchConfiguration(ctx, testFs, ss, simpleSearchPath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := ss.GetSearchRecordByPath(simpleSearchPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that files were parsed
	if len(before.ParsedFiles) == 0 {
		t.Fatal("expected parsed files, got none")
	}

	// Check that both config.tfquery.hcl and variables.tfquery.hcl were parsed
	configFile := ast.SearchFilename("config.tfquery.hcl")
	variablesFile := ast.SearchFilename("variables.tfquery.hcl")

	if _, exists := before.ParsedFiles[configFile]; !exists {
		t.Fatal("expected config.tfquery.hcl to be parsed")
	}

	if _, exists := before.ParsedFiles[variablesFile]; !exists {
		t.Fatal("expected variables.tfquery.hcl to be parsed")
	}

	// ignore job state for next test
	ctx = job.WithIgnoreState(ctx, true)

	// Test single file change parsing (simulating didChange request)
	configURI, err := filepath.Abs(filepath.Join(simpleSearchPath, "config.tfquery.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	changeCtx := lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Search.String(),
		URI:        uri.FromPath(configURI),
	})

	err = ParseSearchConfiguration(changeCtx, testFs, ss, simpleSearchPath)
	if err != nil {
		t.Fatal(err)
	}

	after, err := ss.GetSearchRecordByPath(simpleSearchPath)
	if err != nil {
		t.Fatal(err)
	}

	// Test that config.tfquery.hcl was re-parsed (pointer should be different)
	if before.ParsedFiles[configFile] == after.ParsedFiles[configFile] {
		t.Fatal("config.tfquery.hcl should have been re-parsed")
	}

	// Test that variables.tfquery.hcl was not re-parsed (pointer should be the same)
	if before.ParsedFiles[variablesFile] != after.ParsedFiles[variablesFile] {
		t.Fatal("variables.tfquery.hcl should not have been re-parsed")
	}

	// Verify diagnostics were updated for the changed file
	beforeDiags, beforeOk := before.Diagnostics[globalAst.HCLParsingSource][configFile]
	afterDiags, afterOk := after.Diagnostics[globalAst.HCLParsingSource][configFile]

	if !beforeOk || !afterOk {
		t.Fatal("expected diagnostics for config.tfquery.hcl")
	}

	// The diagnostic objects should be different instances even if content is the same
	if &beforeDiags == &afterDiags {
		t.Fatal("diagnostics should have been updated for config.tfquery.hcl")
	}
}
