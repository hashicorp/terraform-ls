// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/ast"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func TestParsePolicyConfiguration(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	ps, err := state.NewPolicyTestStore(gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	testFs := filesystem.NewFilesystem(gs.DocumentStore)

	simplePolicyTestPath := filepath.Join(testData, "simple-policytest")

	err = ps.Add(simplePolicyTestPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParsePolicyTestConfiguration(ctx, testFs, ps, simplePolicyTestPath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := ps.PolicyTestRecordByPath(simplePolicyTestPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(before.ParsedPolicyTestFiles) == 0 {
		t.Fatal("expected parsed policy files, got none")
	}

	// Verify specific files are in the store
	mainFile := ast.PolicyTestFilename("config.policytest.hcl")

	if _, exists := before.ParsedPolicyTestFiles[mainFile]; !exists {
		t.Fatalf("expected %s to be parsed", mainFile)
	}

	// ignore job state for next test
	ctx = job.WithIgnoreState(ctx, true)

	mainURI, err := filepath.Abs(filepath.Join(simplePolicyTestPath, "config.policytest.hcl"))
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a didChange request for one file
	changeCtx := lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Policy.String(),
		URI:        uri.FromPath(mainURI),
	})

	err = ParsePolicyTestConfiguration(changeCtx, testFs, ps, simplePolicyTestPath)
	if err != nil {
		t.Fatal(err)
	}

	after, err := ps.PolicyTestRecordByPath(simplePolicyTestPath)
	if err != nil {
		t.Fatal(err)
	}

	// config.policy.hcl should have been updated (new pointer)
	if before.ParsedPolicyTestFiles[mainFile] == after.ParsedPolicyTestFiles[mainFile] {
		t.Errorf("%s should have been re-parsed (new pointer expected)", mainFile)
	}

	// Verify diagnostics were updated for the changed file
	if _, ok := after.PolicyTestDiagnostics[globalAst.HCLParsingSource][mainFile]; !ok {
		t.Fatal("expected diagnostics for config.policy.hcl")
	}
}
