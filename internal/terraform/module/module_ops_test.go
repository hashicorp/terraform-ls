package module

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfregistry "github.com/hashicorp/terraform-schema/registry"
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
	err = ParseModuleConfiguration(fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	regClient := registry.NewClient()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/versions" {
			w.Write([]byte(moduleVersionsMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/0.0.8" {
			w.Write([]byte(moduleDataMockResponse))
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
	if diff := cmp.Diff(expectedModuleData, meta, ctydebug.CmpOptions); diff != "" {
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
	err = ParseModuleConfiguration(fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	regClient := registry.NewClient()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/versions" {
			w.Write([]byte(moduleVersionsMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/0.0.8" {
			w.Write([]byte(moduleDataMockResponse))
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
	if diff := cmp.Diff(expectedModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
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
	err = ParseModuleConfiguration(fs, ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ss.Modules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	regClient := registry.NewClient()
	regClient.Timeout = 500 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/versions" {
			w.Write([]byte(moduleVersionsMockResponse))
			return
		}
		if r.RequestURI == "/v1/modules/puppetlabs/deployment/ec/0.0.8" {
			w.Write([]byte(moduleDataMockResponse))
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
	if diff := cmp.Diff(expectedModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}
}

var expectedModuleData = &tfregistry.ModuleData{
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

	err = ParseProviderVersions(fs, ss.Modules, modPath)
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
