// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/state"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// should we expose this from internal/filesystem/filesystem.go instead?
type osFs struct{}

func (osfs osFs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (osfs osFs) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (osfs osFs) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (osfs osFs) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func TestParseTerraformSources(t *testing.T) {
	modPath := t.TempDir()
	manifestDir := filepath.Join(modPath, ".terraform", "modules")
	err := os.MkdirAll(manifestDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(manifestDir, "terraform-sources.json"), []byte(`{
		"terraform_source_bundle": 1,
		"packages": [
		  {
			"source": "git::https://github.com/shernandez5/terraform-kubernetes-crd-demo-module?ref=f6cf642c8671262aac30f0af6e62b6ee85a54204",
			"local": "UmN8ypf1BrY_efIIl4pzoutkgPaJClzCskrS6IWxDfI",
			"meta": {}
		  },
		  {
			"source": "git::https://github.com/shernandez5/terraforming-stacks.git",
			"local": "m9Di4tJSWWxjddtdLEPk1u9uhAdx6uuzJWzUIds_1BQ",
			"meta": {}
		  }
		],
		"registry": [
		  {
			"source": "registry.terraform.io/shernandez5/crd-demo-module/kubernetes",
			"versions": {
			  "0.1.0": {
				"source": "git::https://github.com/shernandez5/terraform-kubernetes-crd-demo-module?ref=f6cf642c8671262aac30f0af6e62b6ee85a54204",
				"deprecation": null
			  }
			}
		  }
		]
	  }`), 0755)

	if err != nil {
		t.Fatal(err)
	}

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = ParseTerraformSources(ctx, osFs{}, rs, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := rs.RootRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	if mod.TerraformSourcesState != operation.OpStateLoaded {
		t.Fatalf("expected state to be loaded, %q given", mod.TerraformSourcesState)
	}

	if mod.TerraformSourcesErr != nil {
		t.Fatalf("unexpected error: %s", mod.TerraformSourcesErr)
	}

	expectedInstalledModules := state.InstalledModules{
		"git::https://github.com/shernandez5/terraform-kubernetes-crd-demo-module?ref=f6cf642c8671262aac30f0af6e62b6ee85a54204": filepath.FromSlash(".terraform/modules/UmN8ypf1BrY_efIIl4pzoutkgPaJClzCskrS6IWxDfI"),
		"git::https://github.com/shernandez5/terraforming-stacks.git":                                                           filepath.FromSlash(".terraform/modules/m9Di4tJSWWxjddtdLEPk1u9uhAdx6uuzJWzUIds_1BQ"),
		"registry.terraform.io/shernandez5/crd-demo-module/kubernetes":                                                          filepath.FromSlash(".terraform/modules/UmN8ypf1BrY_efIIl4pzoutkgPaJClzCskrS6IWxDfI"),
	}
	if diff := cmp.Diff(expectedInstalledModules, mod.InstalledModules); diff != "" {
		t.Fatalf("unexpected installed modules: %s", diff)
	}
}

func TestParseTerraformSources_no_sources_file(t *testing.T) {
	modPath := t.TempDir()
	manifestDir := filepath.Join(modPath, ".terraform", "modules")
	err := os.MkdirAll(manifestDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// not writing any sources file

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = ParseTerraformSources(ctx, osFs{}, rs, modPath)
	if err == nil {
		t.Fatal("expected error for missing sources file")
	}

	mod, err := rs.RootRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	if mod.TerraformSourcesState != operation.OpStateLoaded {
		t.Fatalf("expected state to be loaded, %q given", mod.TerraformSourcesState)
	}

	if mod.TerraformSourcesErr == nil {
		t.Fatal("expected error for missing sources file")
	}

	if !errors.Is(mod.TerraformSourcesErr, os.ErrNotExist) {
		t.Fatalf("unexpected error: %s", mod.TerraformSourcesErr)
	}
}

func TestParseTerraformSources_invalid_sources_file(t *testing.T) {
	modPath := t.TempDir()
	manifestDir := filepath.Join(modPath, ".terraform", "modules")
	err := os.MkdirAll(manifestDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(manifestDir, "terraform-sources.json"), []byte(`{
		"terraform_source_bundle": 0
	  }`), 0755)

	if err != nil {
		t.Fatal(err)
	}

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	err = rs.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = ParseTerraformSources(ctx, osFs{}, rs, modPath)
	if err == nil {
		t.Fatal("expected error for invalid sources file")
	}

	mod, err := rs.RootRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	if mod.TerraformSourcesState != operation.OpStateLoaded {
		t.Fatalf("expected state to be loaded, %q given", mod.TerraformSourcesState)
	}

	if mod.TerraformSourcesErr == nil {
		t.Fatal("expected error for invalid sources file")
	}

	if mod.TerraformSourcesErr.Error() != "failed to parse terraform sources: invalid manifest: unsupported format version 0" {
		t.Fatalf("unexpected error: %s", mod.TerraformSourcesErr)
	}

}
