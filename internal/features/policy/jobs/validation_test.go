// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/policy/ast"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/uri"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

// --- Mocks & Helpers ---

type PolicyRootReaderMock struct{}

func (r PolicyRootReaderMock) InstalledModuleCalls(modPath string) (map[string]tfmod.InstalledModuleCall, error) {
	return nil, nil
}
func (r PolicyRootReaderMock) TerraformVersion(modPath string) *version.Version {
	return nil
}
func (r PolicyRootReaderMock) InstalledModulePath(rootPath string, normalizedSource string) (string, bool) {
	return "", false
}

func setupTestEnv(t *testing.T) (*state.PolicyStore, *filesystem.Filesystem, string) {
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	ps, err := state.NewPolicyStore(gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	policyPath := filepath.Join(testData, "simple-policy")
	err = ps.Add(policyPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	return ps, fs, policyPath
}

func TestParsePolicy_FullPolicy(t *testing.T) {
	ps, fs, policyPath := setupTestEnv(t)
	ctx := lsctx.WithDocumentContext(context.Background(), lsctx.Document{
		Method:     "textDocument/didOpen",
		LanguageID: ilsp.Policy.String(),
	})

	// Run Parse
	if err := ParsePolicyConfiguration(ctx, fs, ps, policyPath); err != nil {
		t.Fatalf("Full parse failed: %v", err)
	}

	// Run Validation
	if err := SchemaPolicyValidation(ctx, ps, PolicyRootReaderMock{}, policyPath); err != nil {
		t.Fatalf("Full schema validation failed: %v", err)
	}

	record, _ := ps.PolicyRecordByPath(policyPath)
	if len(record.ParsedPolicyFiles) == 0 {
		t.Error("Expected files to be parsed, but map is empty")
	}
}

func TestParsePolicy_SingleFile(t *testing.T) {
	ps, fs, policyPath := setupTestEnv(t)

	mainFile := "config.policy.hcl"
	absPath, _ := filepath.Abs(filepath.Join(policyPath, mainFile))
	mainURI := uri.FromPath(absPath)

	ctx := lsctx.WithDocumentContext(context.Background(), lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Policy.String(),
		URI:        mainURI,
	})

	if err := ParsePolicyConfiguration(ctx, fs, ps, policyPath); err != nil {
		t.Fatalf("Incremental parse failed: %v", err)
	}

	if err := SchemaPolicyValidation(ctx, ps, PolicyRootReaderMock{}, policyPath); err != nil {
		t.Fatalf("Incremental schema validation failed: %v", err)
	}

	record, _ := ps.PolicyRecordByPath(policyPath)
	filename := ast.PolicyFilename(mainFile)

	if _, ok := record.PolicyDiagnostics[globalAst.SchemaValidationSource][filename]; !ok {
		t.Errorf("Diagnostic for %s missing from store", filename)
	}
}
