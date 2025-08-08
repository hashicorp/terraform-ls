// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io/fs"
	"log"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/hashicorp/go-version"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/search/state"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/job"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

// Mock implementation of ModuleReader for testing
type mockModuleReader struct{}

func (m *mockModuleReader) LocalModuleMeta(path string) (*tfmod.Meta, error) {
	// Return module metadata with provider requirements that match what the search config uses
	randomAddr := tfaddr.MustParseProviderSource("hashicorp/random")
	awsAddr := tfaddr.MustParseProviderSource("hashicorp/aws")
	unknownAddr := tfaddr.MustParseProviderSource("hashicorp/unknown")

	return &tfmod.Meta{
		ProviderRequirements: tfmod.ProviderRequirements{
			randomAddr:  version.MustConstraints(version.NewConstraint("1.0.0")),
			awsAddr:     version.MustConstraints(version.NewConstraint("3.0.0")),
			unknownAddr: version.MustConstraints(version.NewConstraint("5.0.0")),
		},
		ProviderReferences: map[tfmod.ProviderRef]tfaddr.Provider{
			{LocalName: "aws"}:     awsAddr,
			{LocalName: "random"}:  randomAddr,
			{LocalName: "unknown"}: unknownAddr,
		},
	}, nil
}

// Mock implementation for tests that need empty provider requirements
type emptyMockModuleReader struct{}

func (m *emptyMockModuleReader) LocalModuleMeta(path string) (*tfmod.Meta, error) {
	return &tfmod.Meta{
		ProviderRequirements: tfmod.ProviderRequirements{},
		ProviderReferences:   map[tfmod.ProviderRef]tfaddr.Provider{},
	}, nil
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
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	searchPath := "testsearch"

	fs := filesystem.NewFilesystem(gs.DocumentStore)

	err = ss.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseSearchConfiguration(ctx, fs, ss, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	mockReader := &mockModuleReader{}
	err = LoadSearchMetadata(ctx, ss, mockReader, log.Default(), searchPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	// verify schema was loaded
	pAddr := tfaddr.MustParseProviderSource("hashicorp/random")
	vc := version.MustConstraints(version.NewConstraint(">= 1.0.0"))

	// ask for schema for an unrelated path to avoid path-based matching
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
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	searchPath := "testsearch"

	fs := filesystem.NewFilesystem(gs.DocumentStore)

	err = ss.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseSearchConfiguration(ctx, fs, ss, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	mockReader := &mockModuleReader{}
	err = LoadSearchMetadata(ctx, ss, mockReader, log.Default(), searchPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
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
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	searchPath := "testsearch"

	fs := filesystem.NewFilesystem(gs.DocumentStore)

	err = ss.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseSearchConfiguration(ctx, fs, ss, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	mockReader := &mockModuleReader{}
	err = LoadSearchMetadata(ctx, ss, mockReader, log.Default(), searchPath)
	if err != nil {
		t.Fatal(err)
	}

	// first
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	// second - testing search state
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
	if err != nil {
		if !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}) {
			t.Fatal(err)
		}
	}

	ctx = job.WithIgnoreState(ctx, true)
	// third - testing requirement matching
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
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
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	searchPath := "testsearch"

	fs := filesystem.NewFilesystem(gs.DocumentStore)

	err = ss.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseSearchConfiguration(ctx, fs, ss, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	mockReader := &mockModuleReader{}
	err = LoadSearchMetadata(ctx, ss, mockReader, log.Default(), searchPath)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
		if err != nil && !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}) {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		err := PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
		if err != nil && !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}) {
			t.Error(err)
		}
	}()
	wg.Wait()
}

func TestPreloadEmbeddedSchema_noProviderRequirements(t *testing.T) {
	ctx := context.Background()
	dataDir := "data"
	schemasFS := fstest.MapFS{
		dataDir: &fstest.MapFile{Mode: fs.ModeDir},
	}

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	searchPath := "testsearch"

	fs := filesystem.NewFilesystem(gs.DocumentStore)

	err = ss.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseSearchConfiguration(ctx, fs, ss, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	emptyMockReader := &emptyMockModuleReader{}
	err = LoadSearchMetadata(ctx, ss, emptyMockReader, log.Default(), searchPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreloadEmbeddedSchema_invalidSearchPath(t *testing.T) {
	ctx := context.Background()
	dataDir := "data"
	schemasFS := fstest.MapFS{
		dataDir: &fstest.MapFile{Mode: fs.ModeDir},
	}

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}

	// Don't add the search path to the store, causing SearchRecordByPath to fail
	searchPath := "nonexistent"

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
	if err == nil {
		t.Fatal("expected error for invalid search path")
	}
}

func TestPreloadEmbeddedSchema_alreadyHasSchemas(t *testing.T) {
	ctx := context.Background()
	dataDir := "data"
	schemasFS := fstest.MapFS{
		dataDir: &fstest.MapFile{Mode: fs.ModeDir},
	}

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	searchPath := "testsearch"

	fs := filesystem.NewFilesystem(gs.DocumentStore)

	err = ss.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseSearchConfiguration(ctx, fs, ss, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	mockReader := &mockModuleReader{}
	err = LoadSearchMetadata(ctx, ss, mockReader, log.Default(), searchPath)
	if err != nil {
		t.Fatal(err)
	}

	// Pre-load the schema into the store
	pAddr := tfaddr.MustParseProviderSource("hashicorp/aws")
	err = gs.ProviderSchemas.AddPreloadedSchema(pAddr, version.Must(version.NewVersion("3.0.0")), &tfschema.ProviderSchema{})
	if err != nil {
		t.Fatal(err)
	}

	// This should return early since we already have the schema
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreloadEmbeddedSchema_stateLoading(t *testing.T) {
	ctx := context.Background()
	dataDir := "data"
	schemasFS := fstest.MapFS{
		dataDir: &fstest.MapFile{Mode: fs.ModeDir},
	}

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	searchPath := "testsearch"

	fs := filesystem.NewFilesystem(gs.DocumentStore)

	err = ss.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseSearchConfiguration(ctx, fs, ss, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	mockReader := &mockModuleReader{}
	err = LoadSearchMetadata(ctx, ss, mockReader, log.Default(), searchPath)
	if err != nil {
		t.Fatal(err)
	}

	// Set the state to loading manually
	err = ss.SetPreloadEmbeddedSchemaState(searchPath, operation.OpStateLoading)
	if err != nil {
		t.Fatal(err)
	}

	// This should return early with StateNotChangedErr since state is already loading
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
	if err == nil {
		t.Fatal("expected StateNotChangedErr when state is already loading")
	}

	expectedErr := job.StateNotChangedErr{Dir: document.DirHandleFromPath(searchPath)}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected StateNotChangedErr, got: %v", err)
	}
}

func TestPreloadEmbeddedSchema_multipleProviders(t *testing.T) {
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
		dataDir + "/registry.terraform.io/hashicorp/aws":       &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/aws/3.0.0": &fstest.MapFile{Mode: fs.ModeDir},
		dataDir + "/registry.terraform.io/hashicorp/aws/3.0.0/schema.json.gz": &fstest.MapFile{
			Data: gzipCompressBytes(t, []byte(awsSchemaJSON)),
		},
	}

	gs, err := globalState.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	ss, err := state.NewSearchStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	searchPath := "testsearch"

	fs := filesystem.NewFilesystem(gs.DocumentStore)

	err = ss.Add(searchPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseSearchConfiguration(ctx, fs, ss, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	mockReader := &mockModuleReader{}
	err = LoadSearchMetadata(ctx, ss, mockReader, log.Default(), searchPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, searchPath)
	if err != nil {
		t.Fatal(err)
	}

	// verify both schemas were loaded
	randomAddr := tfaddr.MustParseProviderSource("hashicorp/random")
	awsAddr := tfaddr.MustParseProviderSource("hashicorp/aws")
	vc := version.MustConstraints(version.NewConstraint(">= 1.0.0"))

	// Check random provider schema
	s, err := gs.ProviderSchemas.ProviderSchema("unknown-path", randomAddr, vc)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatalf("expected non-nil schema for %s %s", randomAddr, vc)
	}

	// Check AWS provider schema
	awsVC := version.MustConstraints(version.NewConstraint(">= 3.0.0"))
	s, err = gs.ProviderSchemas.ProviderSchema("unknown-path", awsAddr, awsVC)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatalf("expected non-nil schema for %s %s", awsAddr, awsVC)
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

var awsSchemaJSON = `{
	"format_version": "1.0",
	"provider_schemas": {
		"registry.terraform.io/hashicorp/aws": {
			"provider": {
				"version": 0,
				"block": {
					"attributes": {
						"region": {
							"type": "string",
							"description": "AWS region",
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
