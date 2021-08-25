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
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

func TestModuleStore_Add_duplicate(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = s.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Modules.Add(modPath)
	if err == nil {
		t.Fatal("expected error for duplicate entry")
	}
	existsError := &AlreadyExistsError{}
	if !errors.As(err, &existsError) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestModuleStore_ModuleByPath(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = s.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	tfVersion := version.Must(version.NewVersion("1.0.0"))
	err = s.Modules.UpdateTerraformVersion(modPath, tfVersion, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.Modules.ModuleByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedModule := &Module{
		Path:                  modPath,
		TerraformVersion:      tfVersion,
		TerraformVersionState: operation.OpStateLoaded,
	}
	if diff := cmp.Diff(expectedModule, mod); diff != "" {
		t.Fatalf("unexpected module: %s", diff)
	}
}

func TestModuleStore_CallersOfModule(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	alphaManifest := datadir.NewModuleManifest(
		filepath.Join(tmpDir, "alpha"),
		[]datadir.ModuleRecord{
			{
				Key:        "web_server_sg1",
				SourceAddr: "terraform-aws-modules/security-group/aws//modules/http-80",
				VersionStr: "3.10.0",
				Version:    version.Must(version.NewVersion("3.10.0")),
				Dir:        filepath.Join(".terraform", "modules", "web_server_sg", "terraform-aws-security-group-3.10.0", "modules", "http-80"),
			},
			{
				Dir: ".",
			},
			{
				Key:        "local-x",
				SourceAddr: "../nested/submodule",
				Dir:        filepath.Join("..", "nested", "submodule"),
			},
		},
	)
	betaManifest := datadir.NewModuleManifest(
		filepath.Join(tmpDir, "beta"),
		[]datadir.ModuleRecord{
			{
				Dir: ".",
			},
			{
				Key:        "local-foo",
				SourceAddr: "../another/submodule",
				Dir:        filepath.Join("..", "another", "submodule"),
			},
		},
	)
	gammaManifest := datadir.NewModuleManifest(
		filepath.Join(tmpDir, "gamma"),
		[]datadir.ModuleRecord{
			{
				Key:        "web_server_sg2",
				SourceAddr: "terraform-aws-modules/security-group/aws//modules/http-80",
				VersionStr: "3.10.0",
				Version:    version.Must(version.NewVersion("3.10.0")),
				Dir:        filepath.Join(".terraform", "modules", "web_server_sg", "terraform-aws-security-group-3.10.0", "modules", "http-80"),
			},
			{
				Dir: ".",
			},
			{
				Key:        "local-y",
				SourceAddr: "../nested/submodule",
				Dir:        filepath.Join("..", "nested", "submodule"),
			},
		},
	)

	modules := []struct {
		path        string
		modManifest *datadir.ModuleManifest
	}{
		{
			filepath.Join(tmpDir, "alpha"),
			alphaManifest,
		},
		{
			filepath.Join(tmpDir, "beta"),
			betaManifest,
		},
		{
			filepath.Join(tmpDir, "gamma"),
			gammaManifest,
		},
	}
	for _, mod := range modules {
		err := s.Modules.Add(mod.path)
		if err != nil {
			t.Fatal(err)
		}
		err = s.Modules.UpdateModManifest(mod.path, mod.modManifest, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	submodulePath := filepath.Join(tmpDir, "nested", "submodule")
	mods, err := s.Modules.CallersOfModule(submodulePath)
	if err != nil {
		t.Fatal(err)
	}

	expectedModules := []*Module{
		{
			Path:             filepath.Join(tmpDir, "alpha"),
			ModManifest:      alphaManifest,
			ModManifestState: operation.OpStateLoaded,
		},
		{
			Path:             filepath.Join(tmpDir, "gamma"),
			ModManifest:      gammaManifest,
			ModManifestState: operation.OpStateLoaded,
		},
	}

	if diff := cmp.Diff(expectedModules, mods, cmpOpts); diff != "" {
		t.Fatalf("unexpected modules: %s", diff)
	}
}

func TestModuleStore_List(t *testing.T) {
	s, err := NewStateStore()
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
		err := s.Modules.Add(modPath)
		if err != nil {
			t.Fatal(err)
		}
	}

	modules, err := s.Modules.List()
	if err != nil {
		t.Fatal(err)
	}

	expectedModules := []*Module{
		{Path: filepath.Join(tmpDir, "alpha")},
		{Path: filepath.Join(tmpDir, "beta")},
		{Path: filepath.Join(tmpDir, "gamma")},
	}

	if diff := cmp.Diff(expectedModules, modules, cmpOpts); diff != "" {
		t.Fatalf("unexpected modules: %s", diff)
	}
}

func TestModuleStore_UpdateMetadata(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	metadata := &tfmod.Meta{
		Path:             tmpDir,
		CoreRequirements: testConstraint(t, "~> 0.15"),
		ProviderRequirements: map[tfaddr.Provider]version.Constraints{
			tfaddr.NewDefaultProvider("aws"):    testConstraint(t, "1.2.3"),
			tfaddr.NewDefaultProvider("google"): testConstraint(t, ">= 2.0.0"),
		},
		ProviderReferences: map[tfmod.ProviderRef]tfaddr.Provider{
			{LocalName: "aws"}:    tfaddr.NewDefaultProvider("aws"),
			{LocalName: "google"}: tfaddr.NewDefaultProvider("google"),
		},
	}

	err = s.Modules.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Modules.UpdateMetadata(tmpDir, metadata, nil)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.Modules.ModuleByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedModule := &Module{
		Path: tmpDir,
		Meta: ModuleMetadata{
			CoreRequirements: testConstraint(t, "~> 0.15"),
			ProviderRequirements: map[tfaddr.Provider]version.Constraints{
				tfaddr.NewDefaultProvider("aws"):    testConstraint(t, "1.2.3"),
				tfaddr.NewDefaultProvider("google"): testConstraint(t, ">= 2.0.0"),
			},
			ProviderReferences: map[tfmod.ProviderRef]tfaddr.Provider{
				{LocalName: "aws"}:    tfaddr.NewDefaultProvider("aws"),
				{LocalName: "google"}: tfaddr.NewDefaultProvider("google"),
			},
		},
		MetaState: operation.OpStateLoaded,
	}

	if diff := cmp.Diff(expectedModule, mod, cmpOpts); diff != "" {
		t.Fatalf("unexpected module data: %s", diff)
	}
}

func TestModuleStore_UpdateTerraformVersion(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Modules.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	vErr := customErr{}

	err = s.Modules.UpdateTerraformVersion(tmpDir, testVersion(t, "0.12.4"), nil, vErr)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.Modules.ModuleByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedModule := &Module{
		Path:                  tmpDir,
		TerraformVersion:      testVersion(t, "0.12.4"),
		TerraformVersionState: operation.OpStateLoaded,
		TerraformVersionErr:   vErr,
	}
	if diff := cmp.Diff(expectedModule, mod, cmpOpts); diff != "" {
		t.Fatalf("unexpected module data: %s", diff)
	}
}

func TestModuleStore_UpdateParsedModuleFiles(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Modules.Add(tmpDir)
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

	err = s.Modules.UpdateParsedModuleFiles(tmpDir, ast.ModFiles{
		"test.tf": testFile,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.Modules.ModuleByPath(tmpDir)
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

func TestModuleStore_UpdateParsedVarsFiles(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Modules.Add(tmpDir)
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

	err = s.Modules.UpdateParsedVarsFiles(tmpDir, ast.VarsFilesFromMap(map[string]*hcl.File{
		"test.tfvars": testFile,
	}), nil)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.Modules.ModuleByPath(tmpDir)
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

func TestModuleStore_UpdateModuleDiagnostics(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Modules.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	p := hclparse.NewParser()
	_, diags := p.ParseHCL([]byte(`
provider "blah" {
  region = "london"
`), "test.tf")

	err = s.Modules.UpdateModuleDiagnostics(tmpDir, ast.ModDiagsFromMap(map[string]hcl.Diagnostics{
		"test.tf": diags,
	}))

	mod, err := s.Modules.ModuleByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedDiags := ast.ModDiagsFromMap(map[string]hcl.Diagnostics{
		"test.tf": {
			{
				Severity: hcl.DiagError,
				Summary:  "Argument or block definition required",
				Detail:   "An argument or block definition is required here.",
				Subject: &hcl.Range{
					Filename: "test.tf",
					Start: hcl.Pos{
						Line:   4,
						Column: 1,
						Byte:   39,
					},
					End: hcl.Pos{
						Line:   4,
						Column: 1,
						Byte:   39,
					},
				},
			},
		},
	})
	if diff := cmp.Diff(expectedDiags, mod.ModuleDiagnostics, cmpOpts); diff != "" {
		t.Fatalf("unexpected diagnostics: %s", diff)
	}
}

func TestModuleStore_UpdateVarsDiagnostics(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Modules.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	p := hclparse.NewParser()
	_, diags := p.ParseHCL([]byte(`
dev = {
  region = "london"
`), "test.tfvars")

	err = s.Modules.UpdateVarsDiagnostics(tmpDir, ast.VarsDiagsFromMap(map[string]hcl.Diagnostics{
		"test.tfvars": diags,
	}))

	mod, err := s.Modules.ModuleByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedDiags := ast.VarsDiagsFromMap(map[string]hcl.Diagnostics{
		"test.tfvars": {
			{
				Severity: hcl.DiagError,
				Summary:  "Invalid expression",
				Detail:   "Expected the start of an expression, but found an invalid expression token.",
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
	})
	if diff := cmp.Diff(expectedDiags, mod.VarsDiagnostics, cmpOpts); diff != "" {
		t.Fatalf("unexpected diagnostics: %s", diff)
	}
}

func BenchmarkModuleByPath(b *testing.B) {
	s, err := NewStateStore()
	if err != nil {
		b.Fatal(err)
	}

	modPath := b.TempDir()

	err = s.Modules.Add(modPath)
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
	err = s.Modules.UpdateParsedModuleFiles(modPath, mFiles, nil)
	if err != nil {
		b.Fatal(err)
	}
	mDiags := ast.ModDiagsFromMap(diags)
	err = s.Modules.UpdateModuleDiagnostics(modPath, mDiags)
	if err != nil {
		b.Fatal(err)
	}

	expectedMod := &Module{
		Path:               modPath,
		ParsedModuleFiles:  mFiles,
		ModuleParsingState: operation.OpStateLoaded,
		ModuleDiagnostics:  mDiags,
	}

	for n := 0; n < b.N; n++ {
		mod, err := s.Modules.ModuleByPath(modPath)
		if err != nil {
			b.Fatal(err)
		}

		if diff := cmp.Diff(expectedMod, mod, cmpOpts); diff != "" {
			b.Fatalf("unexpected module: %s", diff)
		}
	}
}

type customErr struct{}

func (e customErr) Error() string {
	return "custom test error"
}

func testConstraint(t testOrBench, v string) version.Constraints {
	constraints, err := version.NewConstraint(v)
	if err != nil {
		t.Fatal(err)
	}
	return constraints
}

func testVersion(t testOrBench, v string) *version.Version {
	ver, err := version.NewVersion(v)
	if err != nil {
		t.Fatal(err)
	}
	return ver
}
