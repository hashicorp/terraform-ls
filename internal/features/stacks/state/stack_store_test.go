// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfstack "github.com/hashicorp/terraform-schema/stack"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

var cmpOpts = cmp.Options{
	cmp.AllowUnexported(StackRecord{}),
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

func TestStackStore_Add_duplicate(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStackStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	stackPath := t.TempDir()

	err = s.Add(stackPath)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Add(stackPath)
	if err == nil {
		t.Fatal("expected error for duplicate entry")
	}
	existsError := &globalState.AlreadyExistsError{}
	if !errors.As(err, &existsError) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestStackStore_StackRecordByPath(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStackStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	stackPath := t.TempDir()

	err = s.Add(stackPath)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.StackRecordByPath(stackPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedRecord := &StackRecord{
		path: stackPath,
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

func TestStackStore_List(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStackStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	stackPaths := []string{
		filepath.Join(tmpDir, "alpha"),
		filepath.Join(tmpDir, "beta"),
		filepath.Join(tmpDir, "gamma"),
	}
	for _, stackPath := range stackPaths {
		err := s.Add(stackPath)
		if err != nil {
			t.Fatal(err)
		}
	}

	stacks, err := s.List()
	if err != nil {
		t.Fatal(err)
	}

	expectedRecords := []*StackRecord{
		{
			path: filepath.Join(tmpDir, "alpha"),
			DiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
		{
			path: filepath.Join(tmpDir, "beta"),
			DiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
		{
			path: filepath.Join(tmpDir, "gamma"),
			DiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
	}

	if diff := cmp.Diff(expectedRecords, stacks, cmpOpts); diff != "" {
		t.Fatalf("unexpected records: %s", diff)
	}
}

func TestStackStore_UpdateMetadata(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStackStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	metadata := &tfstack.Meta{
		Path: tmpDir,
		ProviderRequirements: map[string]tfstack.ProviderRequirement{
			"aws":    {Source: tfaddr.MustParseProviderSource("hashicorp/aws"), VersionConstraints: testConstraint(t, "~> 5.7.0")},
			"google": {Source: tfaddr.MustParseProviderSource("hashicorp/random"), VersionConstraints: testConstraint(t, "~> 3.5.1")},
		},
	}

	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.UpdateMetadata(tmpDir, metadata, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.StackRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedRecord := &StackRecord{
		path: tmpDir,
		Meta: StackMetadata{
			ProviderRequirements: map[string]tfstack.ProviderRequirement{
				"aws":    {Source: tfaddr.MustParseProviderSource("hashicorp/aws"), VersionConstraints: testConstraint(t, "~> 5.7.0")},
				"google": {Source: tfaddr.MustParseProviderSource("hashicorp/random"), VersionConstraints: testConstraint(t, "~> 3.5.1")},
			},
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

func TestStackStore_SetTerraformVersion(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStackStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	version := version.Must(version.NewVersion("1.10.0"))

	err = s.SetTerraformVersion(tmpDir, version)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.StackRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedRecord := &StackRecord{
		path:                          tmpDir,
		RequiredTerraformVersion:      version,
		RequiredTerraformVersionState: operation.OpStateLoaded,
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

func TestStackStore_UpdateParsedFiles(t *testing.T) {
	runTestStackStore_UpdateParsedFiles(t, struct{ extension string }{extension: "tfstack.hcl"})
	runTestStackStore_UpdateParsedFiles(t, struct{ extension string }{extension: "tfcomponent.hcl"})
}

func runTestStackStore_UpdateParsedFiles(t *testing.T, tc struct {
	extension string
}) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStackStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
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
variable "blah" {
  type = string
}
`), "variables."+tc.extension)
	if len(diags) > 0 {
		t.Fatal(diags)
	}

	err = s.UpdateParsedFiles(tmpDir, ast.Files{
		ast.StackFilename("variables." + tc.extension): testFile,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.StackRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedParsedFiles := ast.Files{
		ast.StackFilename("variables." + tc.extension): testFile,
	}
	if diff := cmp.Diff(expectedParsedFiles, record.ParsedFiles, cmpOpts); diff != "" {
		t.Fatalf("unexpected parsed files: %s", diff)
	}
}

func TestStackStore_UpdateDiagnostics(t *testing.T) {
	runTestStackStore_UpdateDiagnostics(t, struct{ extension string }{extension: "tfstack.hcl"})
	runTestStackStore_UpdateDiagnostics(t, struct{ extension string }{extension: "tfcomponent.hcl"})
}

func runTestStackStore_UpdateDiagnostics(t *testing.T, tc struct {
	extension string
}) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStackStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
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
variable "blah" {
  type = string
`), "variables."+tc.extension)

	err = s.UpdateDiagnostics(tmpDir, globalAst.HCLParsingSource, ast.DiagnosticsFromMap(map[string]hcl.Diagnostics{
		"variables." + tc.extension: diags,
	}))
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.StackRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedDiags := ast.SourceDiagnostics{
		globalAst.HCLParsingSource: ast.DiagnosticsFromMap(map[string]hcl.Diagnostics{
			"variables." + tc.extension: {
				{
					Severity: hcl.DiagError,
					Summary:  "Unclosed configuration block",
					Detail:   "There is no closing brace for this block before the end of the file. This may be caused by incorrect brace nesting elsewhere in this file.",
					Subject: &hcl.Range{
						Filename: "variables." + tc.extension,
						Start: hcl.Pos{
							Line:   2,
							Column: 17,
							Byte:   17,
						},
						End: hcl.Pos{
							Line:   2,
							Column: 18,
							Byte:   18,
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

func testConstraint(t *testing.T, v string) version.Constraints {
	constraints, err := version.NewConstraint(v)
	if err != nil {
		t.Fatal(err)
	}
	return constraints
}
