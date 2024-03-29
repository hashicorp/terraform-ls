// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package module

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	tfjson "github.com/hashicorp/terraform-json"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	"github.com/hashicorp/terraform-ls/internal/uri"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfregistry "github.com/hashicorp/terraform-schema/registry"
	"github.com/stretchr/testify/mock"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestGetModuleDataFromRegistry_singleModule(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "uninitialized-external-module")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	regClient := registry.NewClient()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/versions" {
			w.Write([]byte(puppetModuleVersionsMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/0.0.8" {
			w.Write([]byte(puppetModuleDataMockResponse))
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %q", r.RequestURI), 400)
	}))
	regClient.BaseURL = srv.URL
	t.Cleanup(srv.Close)

	err = GetModuleDataFromRegistry(ctx, regClient, ss.Modules, ss.RegistryModules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := tfaddr.ParseModuleSource("puppetlabs/deployment/ec")
	if err != nil {
		t.Fatal(err)
	}
	cons := version.MustConstraints(version.NewConstraint("0.0.8"))

	exists, err := ss.RegistryModules.Exists(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}

	meta, err := ss.Modules.RegistryModuleMeta(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(puppetExpectedModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}
}
func TestGetModuleDataFromRegistry_unreliableInputs(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "unreliable-inputs-module")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	regClient := registry.NewClient()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v1/modules/cloudposse/label/null/versions" {
			w.Write([]byte(labelNullModuleVersionsMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/cloudposse/label/null/0.25.0" {
			w.Write([]byte(labelNullModuleDataOldMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/cloudposse/label/null/0.26.0" {
			w.Write([]byte(labelNullModuleDataNewMockResponse))
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %q", r.RequestURI), 400)
	}))
	regClient.BaseURL = srv.URL
	t.Cleanup(srv.Close)

	err = GetModuleDataFromRegistry(ctx, regClient, ss.Modules, ss.RegistryModules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := tfaddr.ParseModuleSource("cloudposse/label/null")
	if err != nil {
		t.Fatal(err)
	}

	oldCons := version.MustConstraints(version.NewConstraint("0.25.0"))
	exists, err := ss.RegistryModules.Exists(addr, oldCons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, oldCons)
	}
	meta, err := ss.Modules.RegistryModuleMeta(addr, oldCons)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(labelNullExpectedOldModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}

	mewCons := version.MustConstraints(version.NewConstraint("0.26.0"))
	exists, err = ss.RegistryModules.Exists(addr, mewCons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, mewCons)
	}
	meta, err = ss.Modules.RegistryModuleMeta(addr, mewCons)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(labelNullExpectedNewModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}
}

func TestGetModuleDataFromRegistry_moduleNotFound(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "uninitialized-multiple-external-modules")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	regClient := registry.NewClient()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/versions" {
			w.Write([]byte(puppetModuleVersionsMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/0.0.8" {
			w.Write([]byte(puppetModuleDataMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/terraform-aws-modules/eks/aws/versions" {
			http.Error(w, `{"errors":["Not Found"]}`, 404)
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %q", r.RequestURI), 400)
	}))
	regClient.BaseURL = srv.URL
	t.Cleanup(srv.Close)

	err = GetModuleDataFromRegistry(ctx, regClient, ss.Modules, ss.RegistryModules, modPath)
	if err == nil {
		t.Fatal("expected module data obtaining to return error")
	}

	// Verify that 2nd module is still cached even if
	// obtaining data for the other one errored out
	addr, err := tfaddr.ParseModuleSource("puppetlabs/deployment/ec")
	if err != nil {
		t.Fatal(err)
	}
	cons := version.MustConstraints(version.NewConstraint("0.0.8"))

	exists, err := ss.RegistryModules.Exists(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}

	meta, err := ss.Modules.RegistryModuleMeta(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(puppetExpectedModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}

	// Verify that the third module is still cached even if
	// it returns a not found error
	addr, err = tfaddr.ParseModuleSource("terraform-aws-modules/eks/aws")
	if err != nil {
		t.Fatal(err)
	}
	cons = version.MustConstraints(version.NewConstraint("0.0.8"))

	exists, err = ss.RegistryModules.Exists(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}

	// But it shouldn't return any module data
	_, err = ss.Modules.RegistryModuleMeta(addr, cons)
	if err == nil {
		t.Fatal("expected module to be not found")
	}
}

func TestGetModuleDataFromRegistry_apiTimeout(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "uninitialized-multiple-external-modules")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	regClient := registry.NewClient()
	regClient.Timeout = 500 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/versions" {
			w.Write([]byte(puppetModuleVersionsMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/0.0.8" {
			w.Write([]byte(puppetModuleDataMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/terraform-aws-modules/eks/aws/versions" {
			// trigger timeout
			time.Sleep(1 * time.Second)
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %q", r.RequestURI), 400)
	}))
	regClient.BaseURL = srv.URL
	t.Cleanup(srv.Close)

	err = GetModuleDataFromRegistry(ctx, regClient, ss.Modules, ss.RegistryModules, modPath)
	if err == nil {
		t.Fatal("expected module data obtaining to return error")
	}

	// Verify that 2nd module is still cached even if
	// obtaining data for the other one timed out

	addr, err := tfaddr.ParseModuleSource("puppetlabs/deployment/ec")
	if err != nil {
		t.Fatal(err)
	}
	cons := version.MustConstraints(version.NewConstraint("0.0.8"))

	exists, err := ss.RegistryModules.Exists(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}

	meta, err := ss.Modules.RegistryModuleMeta(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(puppetExpectedModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}
}

var puppetExpectedModuleData = &tfregistry.ModuleData{
	Version: version.Must(version.NewVersion("0.0.8")),
	Inputs: []tfregistry.Input{
		{
			Name:        "autoscale",
			Type:        cty.String,
			Default:     cty.StringVal("true"),
			Description: lang.Markdown("Enable autoscaling of elasticsearch"),
			Required:    false,
		},
		{
			Name:        "ec_stack_version",
			Type:        cty.String,
			Default:     cty.StringVal(""),
			Description: lang.Markdown("Version of Elastic Cloud stack to deploy"),
			Required:    false,
		},
		{
			Name:        "name",
			Type:        cty.String,
			Default:     cty.StringVal("ecproject"),
			Description: lang.Markdown("Name of resources"),
			Required:    false,
		},
		{
			Name:        "traffic_filter_sourceip",
			Type:        cty.String,
			Default:     cty.StringVal(""),
			Description: lang.Markdown("traffic filter source IP"),
			Required:    false,
		},
		{
			Name:        "ec_region",
			Type:        cty.String,
			Default:     cty.StringVal("gcp-us-west1"),
			Description: lang.Markdown("cloud provider region"),
			Required:    false,
		},
		{
			Name:        "deployment_templateid",
			Type:        cty.String,
			Default:     cty.StringVal("gcp-io-optimized"),
			Description: lang.Markdown("ID of Elastic Cloud deployment type"),
			Required:    false,
		},
	},
	Outputs: []tfregistry.Output{
		{
			Name:        "elasticsearch_password",
			Description: lang.Markdown("elasticsearch password"),
		},
		{
			Name:        "deployment_id",
			Description: lang.Markdown("Elastic Cloud deployment ID"),
		},
		{
			Name:        "elasticsearch_version",
			Description: lang.Markdown("Stack version deployed"),
		},
		{
			Name:        "elasticsearch_cloud_id",
			Description: lang.Markdown("Elastic Cloud project deployment ID"),
		},
		{
			Name:        "elasticsearch_https_endpoint",
			Description: lang.Markdown("elasticsearch https endpoint"),
		},
		{
			Name:        "elasticsearch_username",
			Description: lang.Markdown("elasticsearch username"),
		},
	},
}

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

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = ParseProviderVersions(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := ss.Modules.ModuleByPath(modPath)
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

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss.SetLogger(log.Default())

	ctx := context.Background()

	err = ss.Modules.Add(modPathFirst)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPathFirst)
	if err != nil {
		t.Fatal(err)
	}
	// parse requirements first to enable schema obtaining later
	err = LoadModuleMetadata(ctx, ss.Modules, modPathFirst)
	if err != nil {
		t.Fatal(err)
	}
	err = ParseProviderVersions(ctx, fs, ss.Modules, modPathFirst)
	if err != nil {
		t.Fatal(err)
	}

	err = ss.Modules.Add(modPathSecond)
	if err != nil {
		t.Fatal(err)
	}
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPathSecond)
	if err != nil {
		t.Fatal(err)
	}
	// parse requirements first to enable schema obtaining later
	err = LoadModuleMetadata(ctx, ss.Modules, modPathSecond)
	if err != nil {
		t.Fatal(err)
	}
	err = ParseProviderVersions(ctx, fs, ss.Modules, modPathSecond)
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

	err = ObtainSchema(ctx, ss.Modules, ss.ProviderSchemas, modPathFirst)
	if err != nil {
		t.Fatal(err)
	}
	err = ObtainSchema(ctx, ss.Modules, ss.ProviderSchemas, modPathSecond)
	if err != nil {
		t.Fatal(err)
	}

	pAddr := tfaddr.MustParseProviderSource("hashicorp/aws")
	vc := version.MustConstraints(version.NewConstraint(">= 4.25.0"))

	// ask for schema for an unrelated module to avoid path-based matching
	s, err := ss.ProviderSchemas.ProviderSchema("third", pAddr, vc)
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

func TestPreloadEmbeddedSchema_basic(t *testing.T) {
	ctx := context.Background()
	dataDir := "data"
	schemasFS := fstest.MapFS{
		dataDir:                            &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io": &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp":              &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/random":       &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/random/1.0.0": &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/random/1.0.0/schema.json.gz": &fstest.MapFile{
			Data: gzipCompressBytes(t, []byte(randomSchemaJSON)),
		},
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	modPath := "testmod"

	cfgFS := fstest.MapFS{
		// These are somewhat awkward double entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		modPath + "/main.tf": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(modPath, "main.tf"): &fstest.MapFile{
			Data: []byte(`terraform {
	required_providers {
		random = {
			source = "hashicorp/random"
			version = "1.0.0"
		}
	}
}
`),
		},
	}

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, cfgFS, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	// verify schema was loaded
	pAddr := tfaddr.MustParseProviderSource("hashicorp/random")
	vc := version.MustConstraints(version.NewConstraint(">= 1.0.0"))

	// ask for schema for an unrelated module to avoid path-based matching
	s, err := ss.ProviderSchemas.ProviderSchema("unknown-path", pAddr, vc)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatalf("expected non-nil schema for %s %s", pAddr, vc)
	}

	_, ok := s.Provider.Attributes["test"]
	if !ok {
		t.Fatalf("expected test attribute in provider schema, not found")
	}
}

func TestPreloadEmbeddedSchema_unknownProviderOnly(t *testing.T) {
	ctx := context.Background()
	dataDir := "data"
	schemasFS := fstest.MapFS{
		dataDir: &fstest.MapFile{Mode: fs.ModeDir},
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	modPath := "testmod"

	cfgFS := fstest.MapFS{
		// These are somewhat awkward double entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		modPath + "/main.tf": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(modPath, "main.tf"): &fstest.MapFile{
			Data: []byte(`terraform {
	required_providers {
		unknown = {
			source = "hashicorp/unknown"
			version = "1.0.0"
		}
	}
}
`),
		},
	}

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, cfgFS, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreloadEmbeddedSchema_idempotency(t *testing.T) {
	ctx := context.Background()
	dataDir := "data"
	schemasFS := fstest.MapFS{
		dataDir:                            &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io": &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp":              &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/random":       &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/random/1.0.0": &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/random/1.0.0/schema.json.gz": &fstest.MapFile{
			Data: gzipCompressBytes(t, []byte(randomSchemaJSON)),
		},
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	modPath := "testmod"

	cfgFS := fstest.MapFS{
		// These are somewhat awkward two entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		modPath + "/main.tf": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(modPath, "main.tf"): &fstest.MapFile{
			Data: []byte(`terraform {
	required_providers {
		random = {
			source = "hashicorp/random"
			version = "1.0.0"
		}
		unknown = {
			source = "hashicorp/unknown"
			version = "5.0.0"
		}
	}
}
`),
		},
	}

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, cfgFS, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	// first
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	// second - testing module state
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		if !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}) {
			t.Fatal(err)
		}
	}

	ctx = job.WithIgnoreState(ctx, true)
	// third - testing requirement matching
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreloadEmbeddedSchema_raceCondition(t *testing.T) {
	ctx := context.Background()
	dataDir := "data"
	schemasFS := fstest.MapFS{
		dataDir:                            &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io": &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp":              &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/random":       &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/random/1.0.0": &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/random/1.0.0/schema.json.gz": &fstest.MapFile{
			Data: gzipCompressBytes(t, []byte(randomSchemaJSON)),
		},
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	modPath := "testmod"

	cfgFS := fstest.MapFS{
		// These are somewhat awkward two entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		modPath + "/main.tf": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(modPath, "main.tf"): &fstest.MapFile{
			Data: []byte(`terraform {
	required_providers {
		random = {
			source = "hashicorp/random"
			version = "1.0.0"
		}
		unknown = {
			source = "hashicorp/unknown"
			version = "5.0.0"
		}
	}
}
`),
		},
	}

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, cfgFS, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss.Modules, ss.ProviderSchemas, modPath)
		if err != nil && !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}) {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		err := PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss.Modules, ss.ProviderSchemas, modPath)
		if err != nil && !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}) {
			t.Error(err)
		}
	}()
	wg.Wait()
}

func TestParseModuleConfiguration(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	testFs := filesystem.NewFilesystem(ss.DocumentStore)

	singleFileModulePath := filepath.Join(testData, "single-file-change-module")

	err = ss.Modules.Add(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, testFs, ss.Modules, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := ss.Modules.ModuleByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// ignore job state
	ctx = job.WithIgnoreState(ctx, true)

	// say we're coming from did_change request
	fooURI, err := filepath.Abs(filepath.Join(singleFileModulePath, "foo.tf"))
	if err != nil {
		t.Fatal(err)
	}
	x := lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Terraform.String(),
		URI:        uri.FromPath(fooURI),
	}
	ctx = lsctx.WithDocumentContext(ctx, x)
	err = ParseModuleConfiguration(ctx, testFs, ss.Modules, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	after, err := ss.Modules.ModuleByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// test if foo.tf is not the same as first seen
	if before.ParsedModuleFiles["foo.tf"] == after.ParsedModuleFiles["foo.tf"] {
		t.Fatal("file should mismatch")
	}

	// test if main.tf is the same as first seen
	if before.ParsedModuleFiles["main.tf"] != after.ParsedModuleFiles["main.tf"] {
		t.Fatal("file mismatch")
	}

	// examine diags should change for foo.tf
	if before.ModuleDiagnostics[ast.HCLParsingSource]["foo.tf"][0] == after.ModuleDiagnostics[ast.HCLParsingSource]["foo.tf"][0] {
		t.Fatal("diags should mismatch")
	}

	// examine diags should change for main.tf
	if before.ModuleDiagnostics[ast.HCLParsingSource]["main.tf"][0] != after.ModuleDiagnostics[ast.HCLParsingSource]["main.tf"][0] {
		t.Fatal("diags should match")
	}
}

func TestParseModuleConfiguration_ignore_tfvars(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	testFs := filesystem.NewFilesystem(ss.DocumentStore)

	singleFileModulePath := filepath.Join(testData, "single-file-change-module")

	err = ss.Modules.Add(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, testFs, ss.Modules, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := ss.Modules.ModuleByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// ignore job state
	ctx = job.WithIgnoreState(ctx, true)

	// say we're coming from did_change request
	fooURI, err := filepath.Abs(filepath.Join(singleFileModulePath, "example.tfvars"))
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Tfvars.String(),
		URI:        uri.FromPath(fooURI),
	})
	err = ParseModuleConfiguration(ctx, testFs, ss.Modules, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	after, err := ss.Modules.ModuleByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// example.tfvars should be missing
	_, beforeExists := before.ParsedModuleFiles["example.tfvars"]
	if beforeExists {
		t.Fatal("example.tfvars file should be missing")
	}
	_, afterExists := after.ParsedModuleFiles["example.tfvars"]
	if afterExists {
		t.Fatal("example.tfvars file should be missing")
	}

	// diags should be missing for example.tfvars
	if _, ok := before.ModuleDiagnostics[ast.HCLParsingSource]["example.tfvars"]; ok {
		t.Fatal("there should be no diags for tfvars files right now")
	}
}

func TestParseVariables(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	testFs := filesystem.NewFilesystem(ss.DocumentStore)

	singleFileModulePath := filepath.Join(testData, "single-file-change-module")

	err = ss.Modules.Add(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, testFs, ss.Modules, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}
	err = ParseVariables(ctx, testFs, ss.Modules, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := ss.Modules.ModuleByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// ignore job state
	ctx = job.WithIgnoreState(ctx, true)

	// say we're coming from did_change request
	filePath, err := filepath.Abs(filepath.Join(singleFileModulePath, "example.tfvars"))
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Tfvars.String(),
		URI:        uri.FromPath(filePath),
	})
	err = ParseVariables(ctx, testFs, ss.Modules, singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	after, err := ss.Modules.ModuleByPath(singleFileModulePath)
	if err != nil {
		t.Fatal(err)
	}

	// example.tfvars should not be the same as first seen
	if before.ParsedVarsFiles["example.tfvars"] == after.ParsedVarsFiles["example.tfvars"] {
		t.Fatal("file should mismatch")
	}

	beforeDiags := before.VarsDiagnostics[ast.HCLParsingSource]
	afterDiags := after.VarsDiagnostics[ast.HCLParsingSource]

	// diags should change for example.tfvars
	if beforeDiags[ast.VarsFilename("example.tfvars")][0] == afterDiags[ast.VarsFilename("example.tfvars")][0] {
		t.Fatal("diags should mismatch")
	}

	if before.ParsedVarsFiles["nochange.tfvars"] != after.ParsedVarsFiles["nochange.tfvars"] {
		t.Fatal("unchanged file should match")
	}

	if beforeDiags[ast.VarsFilename("nochange.tfvars")][0] != afterDiags[ast.VarsFilename("nochange.tfvars")][0] {
		t.Fatal("diags should match for unchanged file")
	}
}

func gzipCompressBytes(t *testing.T, b []byte) []byte {
	var compressedBytes bytes.Buffer
	gw := gzip.NewWriter(&compressedBytes)
	_, err := gw.Write(b)
	if err != nil {
		t.Fatal(err)
	}
	err = gw.Close()
	if err != nil {
		t.Fatal(err)
	}
	return compressedBytes.Bytes()
}

var randomSchemaJSON = `{
	"format_version": "1.0",
	"provider_schemas": {
		"registry.terraform.io/hashicorp/random": {
			"provider": {
				"version": 0,
				"block": {
					"attributes": {
						"test": {
							"type": "string",
							"description": "Test description",
							"description_kind": "markdown",
							"optional": true
						}
					},
					"description_kind": "plain"
				}
			}
		}
	}
}`

func TestSchemaModuleValidation_FullModule(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "invalid-config")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didOpen",
		LanguageID: ilsp.Terraform.String(),
		URI:        "file:///test/variables.tf",
	})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaModuleValidation(ctx, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := ss.Modules.ModuleByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 5
	diagsCount := mod.ModuleDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}

func TestSchemaModuleValidation_SingleFile(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "invalid-config")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Terraform.String(),
		URI:        "file:///test/variables.tf",
	})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaModuleValidation(ctx, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := ss.Modules.ModuleByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 3
	diagsCount := mod.ModuleDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}

func TestSchemaVarsValidation_FullModule(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "invalid-tfvars")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didOpen",
		LanguageID: ilsp.Tfvars.String(),
		URI:        "file:///test/terraform.tfvars",
	})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = ParseVariables(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaVariablesValidation(ctx, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := ss.Modules.ModuleByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 2
	diagsCount := mod.VarsDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}

func TestSchemaVarsValidation_SingleFile(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "invalid-tfvars")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	filePath, err := filepath.Abs(filepath.Join(modPath, "terraform.tfvars"))
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{
		Method:     "textDocument/didChange",
		LanguageID: ilsp.Tfvars.String(),
		URI:        uri.FromPath(filePath),
	})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = ParseVariables(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaVariablesValidation(ctx, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := ss.Modules.ModuleByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 1
	diagsCount := mod.VarsDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}

func TestSchemaVarsValidation_outsideOfModule(t *testing.T) {
	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "standalone-tfvars")

	err = ss.Modules.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(ss.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = ParseVariables(ctx, fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SchemaVariablesValidation(ctx, ss.Modules, ss.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := ss.Modules.ModuleByPath(modPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedCount := 0
	diagsCount := mod.VarsDiagnostics[ast.SchemaValidationSource].Count()
	if diagsCount != expectedCount {
		t.Fatalf("expected %d diagnostics, %d given", expectedCount, diagsCount)
	}
}
