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
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/modules/ast"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

var cmpOpts = cmp.Options{
	cmp.AllowUnexported(ModuleRecord{}),
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

func TestModuleStore_Add_duplicate(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewModuleStore(globalStore.ProviderSchemas, globalStore.RegistryModules, globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = s.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Add(modPath)
	if err == nil {
		t.Fatal("expected error for duplicate entry")
	}
	existsError := &globalState.AlreadyExistsError{}
	if !errors.As(err, &existsError) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestModuleStore_ModuleByPath(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewModuleStore(globalStore.ProviderSchemas, globalStore.RegistryModules, globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = s.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.ModuleRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedModule := &ModuleRecord{
		path: modPath,
		ModuleDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          operation.OpStateUnknown,
			globalAst.SchemaValidationSource:    operation.OpStateUnknown,
			globalAst.ReferenceValidationSource: operation.OpStateUnknown,
			globalAst.TerraformValidateSource:   operation.OpStateUnknown,
		},
	}
	if diff := cmp.Diff(expectedModule, mod, cmpOpts); diff != "" {
		t.Fatalf("unexpected module: %s", diff)
	}
}

func TestModuleStore_List(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewModuleStore(globalStore.ProviderSchemas, globalStore.RegistryModules, globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	modulePaths := []string{
		filepath.Join(tmpDir, "alpha"),
		filepath.Join(tmpDir, "beta"),
		filepath.Join(tmpDir, "gamma"),
	}
	for _, modPath := range modulePaths {
		err := s.Add(modPath)
		if err != nil {
			t.Fatal(err)
		}
	}

	modules, err := s.List()
	if err != nil {
		t.Fatal(err)
	}

	expectedModules := []*ModuleRecord{
		{
			path: filepath.Join(tmpDir, "alpha"),
			ModuleDiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
		{
			path: filepath.Join(tmpDir, "beta"),
			ModuleDiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
		{
			path: filepath.Join(tmpDir, "gamma"),
			ModuleDiagnosticsState: globalAst.DiagnosticSourceState{
				globalAst.HCLParsingSource:          operation.OpStateUnknown,
				globalAst.SchemaValidationSource:    operation.OpStateUnknown,
				globalAst.ReferenceValidationSource: operation.OpStateUnknown,
				globalAst.TerraformValidateSource:   operation.OpStateUnknown,
			},
		},
	}

	if diff := cmp.Diff(expectedModules, modules, cmpOpts); diff != "" {
		t.Fatalf("unexpected modules: %s", diff)
	}
}

func TestModuleStore_UpdateMetadata(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewModuleStore(globalStore.ProviderSchemas, globalStore.RegistryModules, globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	metadata := &tfmod.Meta{
		Path:             tmpDir,
		CoreRequirements: testConstraint(t, "~> 0.15"),
		ProviderRequirements: map[tfaddr.Provider]version.Constraints{
			globalState.NewDefaultProvider("aws"):    testConstraint(t, "1.2.3"),
			globalState.NewDefaultProvider("google"): testConstraint(t, ">= 2.0.0"),
		},
		ProviderReferences: map[tfmod.ProviderRef]tfaddr.Provider{
			{LocalName: "aws"}:    globalState.NewDefaultProvider("aws"),
			{LocalName: "google"}: globalState.NewDefaultProvider("google"),
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

	mod, err := s.ModuleRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedModule := &ModuleRecord{
		path: tmpDir,
		Meta: ModuleMetadata{
			CoreRequirements: testConstraint(t, "~> 0.15"),
			ProviderRequirements: map[tfaddr.Provider]version.Constraints{
				globalState.NewDefaultProvider("aws"):    testConstraint(t, "1.2.3"),
				globalState.NewDefaultProvider("google"): testConstraint(t, ">= 2.0.0"),
			},
			ProviderReferences: map[tfmod.ProviderRef]tfaddr.Provider{
				{LocalName: "aws"}:    globalState.NewDefaultProvider("aws"),
				{LocalName: "google"}: globalState.NewDefaultProvider("google"),
			},
		},
		MetaState: operation.OpStateLoaded,
		ModuleDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          operation.OpStateUnknown,
			globalAst.SchemaValidationSource:    operation.OpStateUnknown,
			globalAst.ReferenceValidationSource: operation.OpStateUnknown,
			globalAst.TerraformValidateSource:   operation.OpStateUnknown,
		},
	}

	if diff := cmp.Diff(expectedModule, mod, cmpOpts); diff != "" {
		t.Fatalf("unexpected module data: %s", diff)
	}
}

func TestModuleStore_UpdateParsedModuleFiles(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewModuleStore(globalStore.ProviderSchemas, globalStore.RegistryModules, globalStore.ChangeStore)
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
provider "blah" {
  region = "london"
}
`), "test.tf")
	if len(diags) > 0 {
		t.Fatal(diags)
	}

	err = s.UpdateParsedModuleFiles(tmpDir, ast.ModFiles{
		"test.tf": testFile,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.ModuleRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedParsedModuleFiles := ast.ModFilesFromMap(map[string]*hcl.File{
		"test.tf": testFile,
	})
	if diff := cmp.Diff(expectedParsedModuleFiles, mod.ParsedModuleFiles, cmpOpts); diff != "" {
		t.Fatalf("unexpected parsed files: %s", diff)
	}
}

func TestModuleStore_UpdateModuleDiagnostics(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewModuleStore(globalStore.ProviderSchemas, globalStore.RegistryModules, globalStore.ChangeStore)
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
provider "blah" {
  region = "london"
`), "test.tf")

	err = s.UpdateModuleDiagnostics(tmpDir, globalAst.HCLParsingSource, ast.ModDiagsFromMap(map[string]hcl.Diagnostics{
		"test.tf": diags,
	}))
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.ModuleRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedDiags := ast.SourceModDiags{
		globalAst.HCLParsingSource: ast.ModDiagsFromMap(map[string]hcl.Diagnostics{
			"test.tf": {
				{
					Severity: hcl.DiagError,
					Summary:  "Unclosed configuration block",
					Detail:   "There is no closing brace for this block before the end of the file. This may be caused by incorrect brace nesting elsewhere in this file.",
					Subject: &hcl.Range{
						Filename: "test.tf",
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
	if diff := cmp.Diff(expectedDiags, mod.ModuleDiagnostics, cmpOpts); diff != "" {
		t.Fatalf("unexpected diagnostics: %s", diff)
	}
}

func TestProviderRequirementsForModule_cycle(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewModuleStore(globalStore.ProviderSchemas, globalStore.RegistryModules, globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	s.MaxModuleNesting = 3

	modHandle := document.DirHandleFromPath(t.TempDir())
	meta := &tfmod.Meta{
		Path: modHandle.Path(),
		ModuleCalls: map[string]tfmod.DeclaredModuleCall{
			"test": {
				LocalName:  "submod",
				SourceAddr: tfmod.LocalSourceAddr("./"),
			},
		},
	}

	err = s.Add(modHandle.Path())
	if err != nil {
		t.Fatal(err)
	}

	err = s.UpdateMetadata(modHandle.Path(), meta, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.ProviderRequirementsForModule(modHandle.Path())
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

func TestProviderRequirementsForModule_basic(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss, err := NewModuleStore(globalStore.ProviderSchemas, globalStore.RegistryModules, globalStore.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	// root module
	modHandle := document.DirHandleFromPath(t.TempDir())
	meta := &tfmod.Meta{
		Path: modHandle.Path(),
		ProviderRequirements: tfmod.ProviderRequirements{
			tfaddr.MustParseProviderSource("hashicorp/aws"): version.MustConstraints(version.NewConstraint(">= 1.0")),
		},
		ModuleCalls: map[string]tfmod.DeclaredModuleCall{
			"test": {
				LocalName:  "submod",
				SourceAddr: tfmod.LocalSourceAddr("./sub"),
			},
		},
	}
	err = ss.Add(modHandle.Path())
	if err != nil {
		t.Fatal(err)
	}
	err = ss.UpdateMetadata(modHandle.Path(), meta, nil)
	if err != nil {
		t.Fatal(err)
	}

	// submodule
	submodHandle := document.DirHandleFromPath(filepath.Join(modHandle.Path(), "sub"))
	subMeta := &tfmod.Meta{
		Path: modHandle.Path(),
		ProviderRequirements: tfmod.ProviderRequirements{
			tfaddr.MustParseProviderSource("hashicorp/google"): version.MustConstraints(version.NewConstraint("> 2.0")),
		},
	}
	err = ss.Add(submodHandle.Path())
	if err != nil {
		t.Fatal(err)
	}
	err = ss.UpdateMetadata(submodHandle.Path(), subMeta, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedReqs := tfmod.ProviderRequirements{
		tfaddr.MustParseProviderSource("hashicorp/aws"):    version.MustConstraints(version.NewConstraint(">= 1.0")),
		tfaddr.MustParseProviderSource("hashicorp/google"): version.MustConstraints(version.NewConstraint("> 2.0")),
	}
	pReqs, err := ss.ProviderRequirementsForModule(modHandle.Path())
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectedReqs, pReqs, cmpOpts); diff != "" {
		t.Fatalf("unexpected requirements: %s", diff)
	}
}

func BenchmarkModuleByPath(b *testing.B) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		b.Fatal(err)
	}
	s, err := NewModuleStore(globalStore.ProviderSchemas, globalStore.RegistryModules, globalStore.ChangeStore)
	if err != nil {
		b.Fatal(err)
	}

	modPath := b.TempDir()

	err = s.Add(modPath)
	if err != nil {
		b.Fatal(err)
	}

	pFiles := make(map[string]*hcl.File, 0)
	diags := make(map[string]hcl.Diagnostics, 0)

	f, pDiags := hclsyntax.ParseConfig([]byte(`provider "blah" {

}
`), "first.tf", hcl.InitialPos)
	diags["first.tf"] = pDiags
	if f != nil {
		pFiles["first.tf"] = f
	}
	f, pDiags = hclsyntax.ParseConfig([]byte(`provider "meh" {


`), "second.tf", hcl.InitialPos)
	diags["second.tf"] = pDiags
	if f != nil {
		pFiles["second.tf"] = f
	}

	mFiles := ast.ModFilesFromMap(pFiles)
	err = s.UpdateParsedModuleFiles(modPath, mFiles, nil)
	if err != nil {
		b.Fatal(err)
	}
	mDiags := ast.ModDiagsFromMap(diags)
	err = s.UpdateModuleDiagnostics(modPath, globalAst.HCLParsingSource, mDiags)
	if err != nil {
		b.Fatal(err)
	}

	expectedMod := &ModuleRecord{
		path:              modPath,
		ParsedModuleFiles: mFiles,
		ModuleDiagnostics: ast.SourceModDiags{
			globalAst.HCLParsingSource: mDiags,
		},
		ModuleDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource: operation.OpStateLoaded,
		},
	}

	for n := 0; n < b.N; n++ {
		mod, err := s.ModuleRecordByPath(modPath)
		if err != nil {
			b.Fatal(err)
		}

		if diff := cmp.Diff(expectedMod, mod, cmpOpts); diff != "" {
			b.Fatalf("unexpected module: %s", diff)
		}
	}
}

type testOrBench interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func testConstraint(t testOrBench, v string) version.Constraints {
	constraints, err := version.NewConstraint(v)
	if err != nil {
		t.Fatal(err)
	}
	return constraints
}
