// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/features/variables/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

var cmpOpts = cmp.Options{
	cmp.AllowUnexported(VariableRecord{}),
	cmp.AllowUnexported(datadir.ModuleManifest{}),
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

func TestVariableStore_Add_duplicate(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewVariableStore(globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	variablePath := t.TempDir()

	err = s.Add(variablePath)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Add(variablePath)
	if err == nil {
		t.Fatal("expected error for duplicate entry")
	}
	existsError := &globalState.AlreadyExistsError{}
	if !errors.As(err, &existsError) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestVariableStore_VariableRecordByPath(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewVariableStore(globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	variablePath := t.TempDir()

	err = s.Add(variablePath)
	if err != nil {
		t.Fatal(err)
	}

	record, err := s.VariableRecordByPath(variablePath)
	if err != nil {
		t.Fatal(err)
	}

	expectedVariable := &VariableRecord{
		path: variablePath,
		VarsDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          operation.OpStateUnknown,
			globalAst.SchemaValidationSource:    operation.OpStateUnknown,
			globalAst.ReferenceValidationSource: operation.OpStateUnknown,
			globalAst.TerraformValidateSource:   operation.OpStateUnknown,
		},
	}
	if diff := cmp.Diff(expectedVariable, record, cmpOpts); diff != "" {
		t.Fatalf("unexpected variable: %s", diff)
	}
}

func TestVariableStore_List(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewVariableStore(globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	variablePaths := []string{
		filepath.Join(tmpDir, "alpha"),
		filepath.Join(tmpDir, "beta"),
		filepath.Join(tmpDir, "gamma"),
	}
	for _, modPath := range variablePaths {
		err := s.Add(modPath)
		if err != nil {
			t.Fatal(err)
		}
	}

	variables, err := s.List()
	if err != nil {
		t.Fatal(err)
	}

	expectedModules := []*VariableRecord{
		{
			path: filepath.Join(tmpDir, "alpha"),
			VarsDiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
		{
			path: filepath.Join(tmpDir, "beta"),
			VarsDiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
		{
			path: filepath.Join(tmpDir, "gamma"),
			VarsDiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
	}

	if diff := cmp.Diff(expectedModules, variables, cmpOpts); diff != "" {
		t.Fatalf("unexpected modules: %s", diff)
	}
}

func TestVariableStore_UpdateParsedVarsFiles(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewVariableStore(globalStore.ChangeStore)
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
dev = {
  region = "london"
}
`), "test.tfvars")
	if len(diags) > 0 {
		t.Fatal(diags)
	}

	err = s.UpdateParsedVarsFiles(tmpDir, ast.VarsFilesFromMap(map[string]*hcl.File{
		"test.tfvars": testFile,
	}), nil)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.VariableRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedParsedVarsFiles := ast.VarsFilesFromMap(map[string]*hcl.File{
		"test.tfvars": testFile,
	})
	if diff := cmp.Diff(expectedParsedVarsFiles, mod.ParsedVarsFiles, cmpOpts); diff != "" {
		t.Fatalf("unexpected parsed files: %s", diff)
	}
}

func TestVariableStore_UpdateVarsDiagnostics(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewVariableStore(globalStore.ChangeStore)
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
dev = {
  region = "london"
`), "test.tfvars")

	err = s.UpdateVarsDiagnostics(tmpDir, globalAst.HCLParsingSource, ast.VarsDiagsFromMap(map[string]hcl.Diagnostics{
		"test.tfvars": diags,
	}))
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.VariableRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedDiags := ast.SourceVarsDiags{
		globalAst.HCLParsingSource: ast.VarsDiagsFromMap(map[string]hcl.Diagnostics{
			"test.tfvars": {
				{
					Severity: hcl.DiagError,
					Summary:  "Missing expression",
					Detail:   "Expected the start of an expression, but found the end of the file.",
					Subject: &hcl.Range{
						Filename: "test.tfvars",
						Start: hcl.Pos{
							Line:   4,
							Column: 1,
							Byte:   29,
						},
						End: hcl.Pos{
							Line:   4,
							Column: 1,
							Byte:   29,
						},
					},
				},
			},
		}),
	}
	if diff := cmp.Diff(expectedDiags, mod.VarsDiagnostics, cmpOpts); diff != "" {
		t.Fatalf("unexpected diagnostics: %s", diff)
	}
}

func TestVariableStore_SetVarsReferenceOriginsState(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewVariableStore(globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	s.SetVarsReferenceOriginsState(tmpDir, operation.OpStateQueued)

	mod, err := s.VariableRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(mod.VarsRefOriginsState, operation.OpStateQueued, cmpOpts); diff != "" {
		t.Fatalf("unexpected module vars ref origins state: %s", diff)
	}
}

func TestVariableStore_UpdateVarsReferenceOrigins(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewVariableStore(globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	origins := reference.Origins{
		reference.PathOrigin{
			Range: hcl.Range{
				Filename: "terraform.tfvars",
				Start: hcl.Pos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
				End: hcl.Pos{
					Line:   1,
					Column: 5,
					Byte:   4,
				},
			},
			TargetAddr: lang.Address{
				lang.RootStep{Name: "var"},
				lang.AttrStep{Name: "name"},
			},
			TargetPath: lang.Path{
				Path:       tmpDir,
				LanguageID: "terraform",
			},
			Constraints: reference.OriginConstraints{
				reference.OriginConstraint{
					OfScopeId: "variable",
					OfType:    cty.String,
				},
			},
		},
	}
	s.UpdateVarsReferenceOrigins(tmpDir, origins, nil)

	mod, err := s.VariableRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(mod.VarsRefOrigins, origins, cmpOpts); diff != "" {
		t.Fatalf("unexpected module vars ref origins: %s", diff)
	}
	if diff := cmp.Diff(mod.VarsRefOriginsState, operation.OpStateLoaded, cmpOpts); diff != "" {
		t.Fatalf("unexpected module vars ref origins state: %s", diff)
	}
}
