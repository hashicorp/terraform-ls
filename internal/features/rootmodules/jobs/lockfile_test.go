// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"io/fs"
	"log"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/features/rootmodules/state"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/stretchr/testify/mock"
	"github.com/zclconf/go-cty/cty"
)

func TestParseProviderVersions(t *testing.T) {
	modPath := "testdir"

	fs := fstest.MapFS{
		modPath: &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join(modPath, ".terraform.lock.hcl"): &fstest.MapFile{
			Data: []byte(`provider "registry.terraform.io/hashicorp/aws" {
  version = "4.23.0"
  hashes = [
    "h1:j6RGCfnoLBpzQVOKUbGyxf4EJtRvQClKplO+WdXL5O0=",
    "zh:17adbedc9a80afc571a8de7b9bfccbe2359e2b3ce1fffd02b456d92248ec9294",
    "zh:23d8956b031d78466de82a3d2bbe8c76cc58482c931af311580b8eaef4e6a38f",
    "zh:343fe19e9a9f3021e26f4af68ff7f4828582070f986b6e5e5b23d89df5514643",
    "zh:6b8ff83d884b161939b90a18a4da43dd464c4b984f54b5f537b2870ce6bd94bc",
    "zh:7777d614d5e9d589ad5508eecf4c6d8f47d50fcbaf5d40fa7921064240a6b440",
    "zh:82f4578861a6fd0cde9a04a1926920bd72d993d524e5b34d7738d4eff3634c44",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:a08fefc153bbe0586389e814979cf7185c50fcddbb2082725991ed02742e7d1e",
    "zh:ae789c0e7cb777d98934387f8888090ccb2d8973ef10e5ece541e8b624e1fb00",
    "zh:b4608aab78b4dbb32c629595797107fc5a84d1b8f0682f183793d13837f0ecf0",
    "zh:ed2c791c2354764b565f9ba4be7fc845c619c1a32cefadd3154a5665b312ab00",
    "zh:f94ac0072a8545eebabf417bc0acbdc77c31c006ad8760834ee8ee5cdb64e743",
  ]
}
`),
		},
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
	err = ParseProviderVersions(ctx, fs, rs, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := rs.RootRecordByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	if mod.InstalledProvidersState != operation.OpStateLoaded {
		t.Fatalf("expected state to be loaded, %q given", mod.InstalledProvidersState)
	}
	expectedInstalledProviders := state.InstalledProviders{
		tfaddr.MustParseProviderSource("hashicorp/aws"): version.Must(version.NewVersion("4.23.0")),
	}
	if diff := cmp.Diff(expectedInstalledProviders, mod.InstalledProviders); diff != "" {
		t.Fatalf("unexpected providers: %s", diff)
	}
}

func TestParseProviderVersions_multipleVersions(t *testing.T) {
	modPathFirst := "first"
	modPathSecond := "second"

	fs := fstest.MapFS{
		modPathFirst: &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join(modPathFirst, ".terraform.lock.hcl"): &fstest.MapFile{
			Data: []byte(`provider "registry.terraform.io/hashicorp/aws" {
  version = "4.23.0"
  hashes = [
    "h1:j6RGCfnoLBpzQVOKUbGyxf4EJtRvQClKplO+WdXL5O0=",
    "zh:17adbedc9a80afc571a8de7b9bfccbe2359e2b3ce1fffd02b456d92248ec9294",
    "zh:23d8956b031d78466de82a3d2bbe8c76cc58482c931af311580b8eaef4e6a38f",
    "zh:343fe19e9a9f3021e26f4af68ff7f4828582070f986b6e5e5b23d89df5514643",
    "zh:6b8ff83d884b161939b90a18a4da43dd464c4b984f54b5f537b2870ce6bd94bc",
    "zh:7777d614d5e9d589ad5508eecf4c6d8f47d50fcbaf5d40fa7921064240a6b440",
    "zh:82f4578861a6fd0cde9a04a1926920bd72d993d524e5b34d7738d4eff3634c44",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:a08fefc153bbe0586389e814979cf7185c50fcddbb2082725991ed02742e7d1e",
    "zh:ae789c0e7cb777d98934387f8888090ccb2d8973ef10e5ece541e8b624e1fb00",
    "zh:b4608aab78b4dbb32c629595797107fc5a84d1b8f0682f183793d13837f0ecf0",
    "zh:ed2c791c2354764b565f9ba4be7fc845c619c1a32cefadd3154a5665b312ab00",
    "zh:f94ac0072a8545eebabf417bc0acbdc77c31c006ad8760834ee8ee5cdb64e743",
  ]
}
`),
		},
		// These are somewhat awkward two entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		modPathFirst + "/main.tf": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(modPathFirst, "main.tf"): &fstest.MapFile{
			Data: []byte(`terraform {
	required_providers {
		aws = {
			source  = "hashicorp/aws"
			version = "4.23.0"
		}
	}
}
`),
		},

		modPathSecond: &fstest.MapFile{Mode: fs.ModeDir},
		filepath.Join(modPathSecond, ".terraform.lock.hcl"): &fstest.MapFile{
			Data: []byte(`provider "registry.terraform.io/hashicorp/aws" {
  version = "4.25.0"
  hashes = [
    "h1:j6RGCfnoLBpzQVOKUbGyxf4EJtRvQClKplO+WdXL5O0=",
    "zh:17adbedc9a80afc571a8de7b9bfccbe2359e2b3ce1fffd02b456d92248ec9294",
    "zh:23d8956b031d78466de82a3d2bbe8c76cc58482c931af311580b8eaef4e6a38f",
    "zh:343fe19e9a9f3021e26f4af68ff7f4828582070f986b6e5e5b23d89df5514643",
    "zh:6b8ff83d884b161939b90a18a4da43dd464c4b984f54b5f537b2870ce6bd94bc",
    "zh:7777d614d5e9d589ad5508eecf4c6d8f47d50fcbaf5d40fa7921064240a6b440",
    "zh:82f4578861a6fd0cde9a04a1926920bd72d993d524e5b34d7738d4eff3634c44",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:a08fefc153bbe0586389e814979cf7185c50fcddbb2082725991ed02742e7d1e",
    "zh:ae789c0e7cb777d98934387f8888090ccb2d8973ef10e5ece541e8b624e1fb00",
    "zh:b4608aab78b4dbb32c629595797107fc5a84d1b8f0682f183793d13837f0ecf0",
    "zh:ed2c791c2354764b565f9ba4be7fc845c619c1a32cefadd3154a5665b312ab00",
    "zh:f94ac0072a8545eebabf417bc0acbdc77c31c006ad8760834ee8ee5cdb64e743",
  ]
}
`),
		},
		// These are somewhat awkward two entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		modPathSecond + "/main.tf": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(modPathSecond, "main.tf"): &fstest.MapFile{
			Data: []byte(`terraform {
	required_providers {
		aws = {
			source = "hashicorp/aws"
			version = "4.25.0"
		}
	}
}
`),
		},
	}

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	rs, err := state.NewRootStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	rs.SetLogger(log.Default())

	ctx := context.Background()

	err = rs.Add(modPathFirst)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	// err = ParseModuleConfiguration(ctx, fs, rs.Modules, modPathFirst)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// // parse requirements first to enable schema obtaining later
	// err = LoadModuleMetadata(ctx, rs.Modules, modPathFirst)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	err = ParseProviderVersions(ctx, fs, rs, modPathFirst)
	if err != nil {
		t.Fatal(err)
	}

	err = rs.Add(modPathSecond)
	if err != nil {
		t.Fatal(err)
	}
	// err = ParseModuleConfiguration(ctx, fs, rs.Modules, modPathSecond)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// // parse requirements first to enable schema obtaining later
	// err = LoadModuleMetadata(ctx, rs.Modules, modPathSecond)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	err = ParseProviderVersions(ctx, fs, rs, modPathSecond)
	if err != nil {
		t.Fatal(err)
	}

	ctx = exec.WithExecutorOpts(ctx, &exec.ExecutorOpts{
		ExecPath: "mock",
	})
	ctx = exec.WithExecutorFactory(ctx, exec.NewMockExecutor(&exec.TerraformMockCalls{
		PerWorkDir: map[string][]*mock.Call{
			"first": {
				{
					Method:        "ProviderSchemas",
					Repeatability: 2,
					Arguments: []interface{}{
						mock.AnythingOfType(""),
					},
					ReturnArguments: []interface{}{
						&tfjson.ProviderSchemas{
							FormatVersion: "1.0",
							Schemas: map[string]*tfjson.ProviderSchema{
								"registry.terraform.io/hashicorp/aws": {
									ConfigSchema: &tfjson.Schema{
										Block: &tfjson.SchemaBlock{
											Attributes: map[string]*tfjson.SchemaAttribute{
												"first": {
													AttributeType: cty.String,
													Optional:      true,
												},
											},
										},
									},
								},
							},
						},
						nil,
					},
				},
			},
			"second": {
				{
					Method:        "ProviderSchemas",
					Repeatability: 2,
					Arguments: []interface{}{
						mock.AnythingOfType(""),
					},
					ReturnArguments: []interface{}{
						&tfjson.ProviderSchemas{
							FormatVersion: "1.0",
							Schemas: map[string]*tfjson.ProviderSchema{
								"registry.terraform.io/hashicorp/aws": {
									ConfigSchema: &tfjson.Schema{
										Block: &tfjson.SchemaBlock{
											Attributes: map[string]*tfjson.SchemaAttribute{
												"second": {
													AttributeType: cty.String,
													Optional:      true,
												},
											},
										},
									},
								},
							},
						},
						nil,
					},
				},
			},
		},
	}))

	err = ObtainSchema(ctx, rs, gs.ProviderSchemas, modPathFirst)
	if err != nil {
		t.Fatal(err)
	}
	err = ObtainSchema(ctx, rs, gs.ProviderSchemas, modPathSecond)
	if err != nil {
		t.Fatal(err)
	}

	pAddr := tfaddr.MustParseProviderSource("hashicorp/aws")
	vc := version.MustConstraints(version.NewConstraint(">= 4.25.0"))

	// ask for schema for an unrelated module to avoid path-based matching
	s, err := gs.ProviderSchemas.ProviderSchema("third", pAddr, vc)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatalf("expected non-nil schema for %s %s", pAddr, vc)
	}

	_, ok := s.Provider.Attributes["second"]
	if !ok {
		t.Fatalf("expected attribute from second provider schema, not found")
	}
}
