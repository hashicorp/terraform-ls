package module

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/state"
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
