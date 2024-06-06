// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

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
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/registry"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfregistry "github.com/hashicorp/terraform-schema/registry"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestGetModuleDataFromRegistry_singleModule(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "uninitialized-external-module")

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ctx, ms, modPath)
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

	err = GetModuleDataFromRegistry(ctx, regClient, ms, gs.RegistryModules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := tfaddr.ParseModuleSource("puppetlabs/deployment/ec")
	if err != nil {
		t.Fatal(err)
	}
	cons := version.MustConstraints(version.NewConstraint("0.0.8"))

	exists, err := gs.RegistryModules.Exists(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}

	meta, err := ms.RegistryModuleMeta(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(puppetExpectedModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}
}

func TestGetModuleDataFromRegistry_unreliableInputs(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "unreliable-inputs-module")

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ctx, ms, modPath)
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

	err = GetModuleDataFromRegistry(ctx, regClient, ms, gs.RegistryModules, modPath)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := tfaddr.ParseModuleSource("cloudposse/label/null")
	if err != nil {
		t.Fatal(err)
	}

	oldCons := version.MustConstraints(version.NewConstraint("0.25.0"))
	exists, err := gs.RegistryModules.Exists(addr, oldCons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, oldCons)
	}
	meta, err := ms.RegistryModuleMeta(addr, oldCons)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(labelNullExpectedOldModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}

	mewCons := version.MustConstraints(version.NewConstraint("0.26.0"))
	exists, err = gs.RegistryModules.Exists(addr, mewCons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, mewCons)
	}
	meta, err = ms.RegistryModuleMeta(addr, mewCons)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(labelNullExpectedNewModuleData, meta, ctydebug.CmpOptions); diff != "" {
		t.Fatalf("metadata mismatch: %s", diff)
	}
}

func TestGetModuleDataFromRegistry_moduleNotFound(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "uninitialized-multiple-external-modules")

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ctx, ms, modPath)
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

	err = GetModuleDataFromRegistry(ctx, regClient, ms, gs.RegistryModules, modPath)
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

	exists, err := gs.RegistryModules.Exists(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}

	meta, err := ms.RegistryModuleMeta(addr, cons)
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

	exists, err = gs.RegistryModules.Exists(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}

	// But it shouldn't return any module data
	_, err = ms.RegistryModuleMeta(addr, cons)
	if err == nil {
		t.Fatal("expected module to be not found")
	}
}

func TestGetModuleDataFromRegistry_apiTimeout(t *testing.T) {
	ctx := context.Background()
	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
	if err != nil {
		t.Fatal(err)
	}

	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(testData, "uninitialized-multiple-external-modules")

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := filesystem.NewFilesystem(gs.DocumentStore)
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, fs, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = LoadModuleMetadata(ctx, ms, modPath)
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

	err = GetModuleDataFromRegistry(ctx, regClient, ms, gs.RegistryModules, modPath)
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

	exists, err := gs.RegistryModules.Exists(addr, cons)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected cached metadata to exist for %q %q", addr, cons)
	}

	meta, err := ms.RegistryModuleMeta(addr, cons)
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

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
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

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, cfgFS, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ms, gs.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	// verify schema was loaded
	pAddr := tfaddr.MustParseProviderSource("hashicorp/random")
	vc := version.MustConstraints(version.NewConstraint(">= 1.0.0"))

	// ask for schema for an unrelated module to avoid path-based matching
	s, err := gs.ProviderSchemas.ProviderSchema("unknown-path", pAddr, vc)
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

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
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

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, cfgFS, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ms, gs.ProviderSchemas, modPath)
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

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
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

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, cfgFS, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}

	// first
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ms, gs.ProviderSchemas, modPath)
	if err != nil {
		t.Fatal(err)
	}

	// second - testing module state
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ms, gs.ProviderSchemas, modPath)
	if err != nil {
		if !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}) {
			t.Fatal(err)
		}
	}

	ctx = job.WithIgnoreState(ctx, true)
	// third - testing requirement matching
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ms, gs.ProviderSchemas, modPath)
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

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := state.NewModuleStore(gs.ProviderSchemas, gs.RegistryModules, gs.ChangeStore)
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

	err = ms.Add(modPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseModuleConfiguration(ctx, cfgFS, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadModuleMetadata(ctx, ms, modPath)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ms, gs.ProviderSchemas, modPath)
		if err != nil && !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}) {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		err := PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ms, gs.ProviderSchemas, modPath)
		if err != nil && !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}) {
			t.Error(err)
		}
	}()
	wg.Wait()
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
