// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfmod "github.com/hashicorp/terraform-schema/module"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

var cmpOpts = cmp.Options{
	cmp.AllowUnexported(RootRecord{}),
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
	s, err := NewRootStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
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
	s, err := NewRootStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	modPath := t.TempDir()

	err = s.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	tfVersion := version.Must(version.NewVersion("1.0.0"))
	err = s.UpdateTerraformAndProviderVersions(modPath, tfVersion, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.RootRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedModule := &RootRecord{
		path:                  modPath,
		TerraformVersion:      tfVersion,
		TerraformVersionState: operation.OpStateLoaded,
	}
	if diff := cmp.Diff(expectedModule, mod, cmpOpts); diff != "" {
		t.Fatalf("unexpected module: %s", diff)
	}
}

func TestModuleStore_CallersOfModule(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewRootStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()

	alphaManifest := datadir.NewModuleManifest(
		filepath.Join(tmpDir, "alpha"),
		[]datadir.ModuleRecord{
			{
				Key:        "web_server_sg1",
				SourceAddr: tfmod.ParseModuleSourceAddr("terraform-aws-modules/security-group/aws//modules/http-80"),
				VersionStr: "3.10.0",
				Version:    version.Must(version.NewVersion("3.10.0")),
				Dir:        filepath.Join(".terraform", "modules", "web_server_sg", "terraform-aws-security-group-3.10.0", "modules", "http-80"),
			},
			{
				Dir: ".",
			},
			{
				Key:        "local-x",
				SourceAddr: tfmod.ParseModuleSourceAddr("../nested/submodule"),
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
				SourceAddr: tfmod.ParseModuleSourceAddr("../another/submodule"),
				Dir:        filepath.Join("..", "another", "submodule"),
			},
		},
	)
	gammaManifest := datadir.NewModuleManifest(
		filepath.Join(tmpDir, "gamma"),
		[]datadir.ModuleRecord{
			{
				Key:        "web_server_sg2",
				SourceAddr: tfmod.ParseModuleSourceAddr("terraform-aws-modules/security-group/aws//modules/http-80"),
				VersionStr: "3.10.0",
				Version:    version.Must(version.NewVersion("3.10.0")),
				Dir:        filepath.Join(".terraform", "modules", "web_server_sg", "terraform-aws-security-group-3.10.0", "modules", "http-80"),
			},
			{
				Dir: ".",
			},
			{
				Key:        "local-y",
				SourceAddr: tfmod.ParseModuleSourceAddr("../nested/submodule"),
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
		err := s.Add(mod.path)
		if err != nil {
			t.Fatal(err)
		}
		err = s.UpdateModManifest(mod.path, mod.modManifest, nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	submodulePath := filepath.Join(tmpDir, "nested", "submodule")
	callers, err := s.CallersOfModule(submodulePath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCallers := []string{
		filepath.Join(tmpDir, "alpha"),
		filepath.Join(tmpDir, "gamma"),
	}

	if diff := cmp.Diff(expectedCallers, callers, cmpOpts); diff != "" {
		t.Fatalf("unexpected modules: %s", diff)
	}
}

func TestModuleStore_List(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewRootStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
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

	expectedModules := []*RootRecord{
		{
			path: filepath.Join(tmpDir, "alpha"),
		},
		{
			path: filepath.Join(tmpDir, "beta"),
		},
		{
			path: filepath.Join(tmpDir, "gamma"),
		},
	}

	if diff := cmp.Diff(expectedModules, modules, cmpOpts); diff != "" {
		t.Fatalf("unexpected modules: %s", diff)
	}
}

func TestModuleStore_UpdateTerraformAndProviderVersions(t *testing.T) {
	globalStore, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewRootStore(globalStore.ChangeStore, globalStore.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	err = s.Add(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	vErr := customErr{}

	err = s.UpdateTerraformAndProviderVersions(tmpDir, testVersion(t, "0.12.4"), nil, vErr)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := s.RootRecordByPath(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedModule := &RootRecord{
		path:                  tmpDir,
		TerraformVersion:      testVersion(t, "0.12.4"),
		TerraformVersionState: operation.OpStateLoaded,
		TerraformVersionErr:   vErr,
	}
	if diff := cmp.Diff(expectedModule, mod, cmpOpts); diff != "" {
		t.Fatalf("unexpected module data: %s", diff)
	}
}

type customErr struct{}

func (e customErr) Error() string {
	return "custom test error"
}

type testOrBench interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func testVersion(t testOrBench, v string) *version.Version {
	ver, err := version.NewVersion(v)
	if err != nil {
		t.Fatal(err)
	}
	return ver
}
