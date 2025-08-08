// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/features/search/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfsearch "github.com/hashicorp/terraform-schema/search"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

var cmpOpts = cmp.Options{
	cmp.AllowUnexported(SearchRecord{}),
	cmp.AllowUnexported(hclsyntax.Body{}),
	cmp.Comparer(func(x, y version.Constraint) bool {
		return x.String() == y.String()
	}),
	cmp.Comparer(func(x, y hcl.File) bool {
		return (x.Body == y.Body &&
			cmp.Equal(x.Bytes, y.Bytes))
	}),
	ctydebug.CmpOptions,
}

func TestSearchStore_Add_duplicate(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	searchPath := t.TempDir()

	err = s.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Add(searchPath)
	if err == nil {
		t.Fatal("expected error for duplicate entry")
	}
	existsError := &globalState.AlreadyExistsError{}
	if !errors.As(err, &existsError) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestSearchStore_SearchRecordByPath(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	searchPath := t.TempDir()

	err = s.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedRecord := &SearchRecord{
		path:                       searchPath,
		PreloadEmbeddedSchemaState: operation.OpStateUnknown,
		RefOriginsState:            operation.OpStateUnknown,
		RefTargetsState:            operation.OpStateUnknown,
		MetaState:                  operation.OpStateUnknown,
		DiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          operation.OpStateUnknown,
			globalAst.SchemaValidationSource:    operation.OpStateUnknown,
			globalAst.ReferenceValidationSource: operation.OpStateUnknown,
			globalAst.TerraformValidateSource:   operation.OpStateUnknown,
		},
	}
	if diff := cmp.Diff(expectedRecord, record, cmpOpts); diff != "" {
		t.Fatalf("unexpected record: %s", diff)
	}
}

func TestSearchStore_List(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	searchPaths := []string{
		filepath.Join(tmpDir, "alpha"),
		filepath.Join(tmpDir, "beta"),
		filepath.Join(tmpDir, "gamma"),
	}
	for _, searchPath := range searchPaths {
		err := s.Add(searchPath)
		if err != nil {
			t.Fatal(err)
		}
	}

	searches, err := s.List()
	if err != nil {
		t.Fatal(err)
	}

	expectedRecords := []*SearchRecord{
		{
			path:                       filepath.Join(tmpDir, "alpha"),
			PreloadEmbeddedSchemaState: operation.OpStateUnknown,
			RefOriginsState:            operation.OpStateUnknown,
			RefTargetsState:            operation.OpStateUnknown,
			MetaState:                  operation.OpStateUnknown,
			DiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
		{
			path:                       filepath.Join(tmpDir, "beta"),
			PreloadEmbeddedSchemaState: operation.OpStateUnknown,
			RefOriginsState:            operation.OpStateUnknown,
			RefTargetsState:            operation.OpStateUnknown,
			MetaState:                  operation.OpStateUnknown,
			DiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
		{
			path:                       filepath.Join(tmpDir, "gamma"),
			PreloadEmbeddedSchemaState: operation.OpStateUnknown,
			RefOriginsState:            operation.OpStateUnknown,
			RefTargetsState:            operation.OpStateUnknown,
			MetaState:                  operation.OpStateUnknown,
			DiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
	}

	if diff := cmp.Diff(expectedRecords, searches, cmpOpts); diff != "" {
		t.Fatalf("unexpected records: %s", diff)
	}
}

func TestSearchStore_Remove(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	searchPath := t.TempDir()

	err = s.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify it exists
	if !s.Exists(searchPath) {
		t.Fatal("expected search to exist before removal")
	}

	err = s.Remove(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify it's removed
	if s.Exists(searchPath) {
		t.Fatal("expected search to be removed")
	}

	// Removing again should not error
	err = s.Remove(searchPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSearchStore_Exists(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	searchPath := t.TempDir()

	// Should not exist initially
	if s.Exists(searchPath) {
		t.Fatal("expected search to not exist initially")
	}

	err = s.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	// Should exist after adding
	if !s.Exists(searchPath) {
		t.Fatal("expected search to exist after adding")
	}
}

func TestSearchStore_AddIfNotExists(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	searchPath := t.TempDir()

	// Should add if not exists
	err = s.AddIfNotExists(searchPath)
	if err != nil {
		t.Fatal(err)
	}

	if !s.Exists(searchPath) {
		t.Fatal("expected search to exist after AddIfNotExists")
	}

	// Should not error if already exists
	err = s.AddIfNotExists(searchPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSearchStore_UpdateMetadata(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	metadata := &tfsearch.Meta{
		Lists: map[string]tfsearch.List{
			"my_list": {},
		},
		Variables: map[string]tfsearch.Variable{
			"my_var": {},
		},
		Filenames: []string{"test.tfquery.hcl"},
		ProviderReferences: map[tfsearch.ProviderRef]tfaddr.Provider{
			{LocalName: "aws"}: tfaddr.MustParseProviderSource("hashicorp/aws"),
		},
		ProviderRequirements: map[tfaddr.Provider]version.Constraints{
			tfaddr.MustParseProviderSource("hashicorp/aws"): testConstraint(t, "~> 5.7.0"),
		},
		CoreRequirements: testConstraint(t, ">= 1.0"),
	}

	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.UpdateMetadata(tmpDir, metadata, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedRecord := &SearchRecord{
		path:                       tmpDir,
		PreloadEmbeddedSchemaState: operation.OpStateUnknown,
		RefOriginsState:            operation.OpStateUnknown,
		RefTargetsState:            operation.OpStateUnknown,
		Meta: SearchMetadata{
			Lists: map[string]tfsearch.List{
				"my_list": {},
			},
			Variables: map[string]tfsearch.Variable{
				"my_var": {},
			},
			Filenames: []string{"test.tfquery.hcl"},
			ProviderReferences: map[tfsearch.ProviderRef]tfaddr.Provider{
				{LocalName: "aws"}: tfaddr.MustParseProviderSource("hashicorp/aws"),
			},
			ProviderRequirements: map[tfaddr.Provider]version.Constraints{
				tfaddr.MustParseProviderSource("hashicorp/aws"): testConstraint(t, "~> 5.7.0"),
			},
			CoreRequirements: testConstraint(t, ">= 1.0"),
		},
		MetaState: operation.OpStateLoaded,
		DiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          operation.OpStateUnknown,
			globalAst.SchemaValidationSource:    operation.OpStateUnknown,
			globalAst.ReferenceValidationSource: operation.OpStateUnknown,
			globalAst.TerraformValidateSource:   operation.OpStateUnknown,
		},
	}

	if diff := cmp.Diff(expectedRecord, record, cmpOpts); diff != "" {
		t.Fatalf("unexpected record data: %s", diff)
	}
}

func TestSearchStore_UpdateParsedFiles(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	p := hclparse.NewParser()
	testFile, diags := p.ParseHCL([]byte(`
variable "test_var" {
  type = string
}

list "resource_search" "main" {
  limit = 10
  include_resource = var.test_var
}
`), "test.tfquery.hcl")
	if len(diags) > 0 {
		t.Fatal(diags)
	}

	err = s.UpdateParsedFiles(tmpDir, ast.Files{
		ast.SearchFilename("test.tfquery.hcl"): testFile,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedParsedFiles := ast.Files{
		ast.SearchFilename("test.tfquery.hcl"): testFile,
	}
	if diff := cmp.Diff(expectedParsedFiles, record.ParsedFiles, cmpOpts); diff != "" {
		t.Fatalf("unexpected parsed files: %s", diff)
	}
}

func TestSearchStore_UpdateDiagnostics(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	p := hclparse.NewParser()
	_, diags := p.ParseHCL([]byte(`
variable "test_var" {
  type = string
`), "test.tfquery.hcl")

	err = s.UpdateDiagnostics(tmpDir, globalAst.HCLParsingSource, ast.DiagnosticsFromMap(map[string]hcl.Diagnostics{
		"test.tfquery.hcl": diags,
	}))
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedDiags := ast.SourceDiagnostics{
		globalAst.HCLParsingSource: ast.DiagnosticsFromMap(map[string]hcl.Diagnostics{
			"test.tfquery.hcl": {
				{
					Severity: hcl.DiagError,
					Summary:  "Unclosed configuration block",
					Detail:   "There is no closing brace for this block before the end of the file. This may be caused by incorrect brace nesting elsewhere in this file.",
					Subject: &hcl.Range{
						Filename: "test.tfquery.hcl",
						Start: hcl.Pos{
							Line:   2,
							Column: 21,
							Byte:   21,
						},
						End: hcl.Pos{
							Line:   2,
							Column: 22,
							Byte:   22,
						},
					},
				},
			},
		}),
	}
	if diff := cmp.Diff(expectedDiags, record.Diagnostics, cmpOpts); diff != "" {
		t.Fatalf("unexpected diagnostics: %s", diff)
	}
}

func TestSearchStore_SetDiagnosticsState(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.SetDiagnosticsState(tmpDir, globalAst.HCLParsingSource, operation.OpStateLoaded)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if record.DiagnosticsState[globalAst.HCLParsingSource] != operation.OpStateLoaded {
		t.Fatalf("expected HCLParsingSource state to be OpStateLoaded, got %v", record.DiagnosticsState[globalAst.HCLParsingSource])
	}
}

func TestSearchStore_SetMetaState(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.SetMetaState(tmpDir, operation.OpStateLoaded)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if record.MetaState != operation.OpStateLoaded {
		t.Fatalf("expected MetaState to be OpStateLoaded, got %v", record.MetaState)
	}
}

func TestSearchStore_SetPreloadEmbeddedSchemaState(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.SetPreloadEmbeddedSchemaState(tmpDir, operation.OpStateLoaded)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if record.PreloadEmbeddedSchemaState != operation.OpStateLoaded {
		t.Fatalf("expected PreloadEmbeddedSchemaState to be OpStateLoaded, got %v", record.PreloadEmbeddedSchemaState)
	}
}

func TestSearchStore_SetReferenceTargetsState(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.SetReferenceTargetsState(tmpDir, operation.OpStateLoaded)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if record.RefTargetsState != operation.OpStateLoaded {
		t.Fatalf("expected RefTargetsState to be OpStateLoaded, got %v", record.RefTargetsState)
	}
}

func TestSearchStore_UpdateReferenceTargets(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Test with empty targets - the actual structure would depend on the reference package
	targets := make(reference.Targets, 0)

	err = s.UpdateReferenceTargets(tmpDir, targets, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(record.RefTargets) != 0 {
		t.Fatalf("expected 0 reference targets, got %d", len(record.RefTargets))
	}

	if record.RefTargetsState != operation.OpStateLoaded {
		t.Fatalf("expected RefTargetsState to be OpStateLoaded, got %v", record.RefTargetsState)
	}
}

func TestSearchStore_SetReferenceOriginsState(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.SetReferenceOriginsState(tmpDir, operation.OpStateLoaded)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if record.RefOriginsState != operation.OpStateLoaded {
		t.Fatalf("expected RefOriginsState to be OpStateLoaded, got %v", record.RefOriginsState)
	}
}

func TestSearchStore_UpdateReferenceOrigins(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Test with empty origins - the actual structure would depend on the reference package
	origins := make(reference.Origins, 0)

	err = s.UpdateReferenceOrigins(tmpDir, origins, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.SearchRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(record.RefOrigins) != 0 {
		t.Fatalf("expected 0 reference origins, got %d", len(record.RefOrigins))
	}

	if record.RefOriginsState != operation.OpStateLoaded {
		t.Fatalf("expected RefOriginsState to be OpStateLoaded, got %v", record.RefOriginsState)
	}
}

func TestSearchStore_ProviderRequirementsForModule(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Update metadata with provider requirements
	metadata := &tfsearch.Meta{
		ProviderRequirements: map[tfaddr.Provider]version.Constraints{
			tfaddr.MustParseProviderSource("hashicorp/aws"):    testConstraint(t, "~> 5.7.0"),
			tfaddr.MustParseProviderSource("hashicorp/random"): testConstraint(t, "~> 3.5.1"),
		},
	}

	err = s.UpdateMetadata(tmpDir, metadata, nil)
	if err != nil {
		t.Fatal(err)
	}

	requirements, err := s.ProviderRequirementsForModule(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedRequirements := tfsearch.ProviderRequirements{
		tfaddr.MustParseProviderSource("hashicorp/aws"):    testConstraint(t, "~> 5.7.0"),
		tfaddr.MustParseProviderSource("hashicorp/random"): testConstraint(t, "~> 3.5.1"),
	}

	if diff := cmp.Diff(expectedRequirements, requirements, cmpOpts); diff != "" {
		t.Fatalf("unexpected provider requirements: %s", diff)
	}
}

func TestSearchStore_ProviderRequirementsForModule_NotFound(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearchStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	// Don't add the module to the store
	tmpDir := t.TempDir()

	requirements, err := s.ProviderRequirementsForModule(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Should return empty requirements when module is not found
	if len(requirements) != 0 {
		t.Fatalf("expected empty provider requirements for non-existent module, got %d", len(requirements))
	}
}

func testConstraint(t *testing.T, v string) version.Constraints {
	constraints, err := version.NewConstraint(v)
	if err != nil {
		t.Fatal(err)
	}
	return constraints
}
