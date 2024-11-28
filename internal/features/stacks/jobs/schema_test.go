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
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/hashicorp/go-version"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/job"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

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
	ss, err := state.NewStackStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	stackPath := "teststack"

	cfgFS := fstest.MapFS{
		// These are somewhat awkward double entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		stackPath + "/providers.tfstack.hcl": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(stackPath, "providers.tfstack.hcl"): &fstest.MapFile{
			Data: []byte(`required_providers {
	random = {
		source = "hashicorp/random"
		version = "1.0.0"
	}
}
`),
		},
	}

	err = ss.Add(stackPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseStackConfiguration(ctx, cfgFS, ss, stackPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadStackMetadata(ctx, ss, stackPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, stackPath)
	if err != nil {
		t.Fatal(err)
	}

	// verify schema was loaded
	pAddr := tfaddr.MustParseProviderSource("hashicorp/random")
	vc := version.MustConstraints(version.NewConstraint(">= 1.0.0"))

	// ask for schema for an unrelated stack to avoid path-based matching
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
	ss, err := state.NewStackStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	stackPath := "teststack"

	cfgFS := fstest.MapFS{
		// These are somewhat awkward double entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		stackPath + "/providers.tfstack.hcl": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(stackPath, "providers.tfstack.hcl"): &fstest.MapFile{
			Data: []byte(`required_providers {
	unknown = {
		source = "hashicorp/unknown"
		version = "1.0.0"
	}
}
`),
		},
	}

	err = ss.Add(stackPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseStackConfiguration(ctx, cfgFS, ss, stackPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadStackMetadata(ctx, ss, stackPath)
	if err != nil {
		t.Fatal(err)
	}

	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, stackPath)
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
	ss, err := state.NewStackStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	stackPath := "teststack"

	cfgFS := fstest.MapFS{
		// These are somewhat awkward two entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		stackPath + "/providers.tfstack.hcl": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(stackPath, "providers.tfstack.hcl"): &fstest.MapFile{
			Data: []byte(`required_providers {
	random = {
		source = "hashicorp/random"
		version = "1.0.0"
	}
	unknown = {
		source = "hashicorp/unknown"
		version = "5.0.0"
	}
}
`),
		},
	}

	err = ss.Add(stackPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseStackConfiguration(ctx, cfgFS, ss, stackPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadStackMetadata(ctx, ss, stackPath)
	if err != nil {
		t.Fatal(err)
	}

	// first
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, stackPath)
	if err != nil {
		t.Fatal(err)
	}

	// second - testing stack state
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, stackPath)
	if err != nil {
		if !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}) {
			t.Fatal(err)
		}
	}

	ctx = job.WithIgnoreState(ctx, true)
	// third - testing requirement matching
	err = PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, stackPath)
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
	ss, err := state.NewStackStore(gs.ChangeStore, gs.ProviderSchemas)
	if err != nil {
		t.Fatal(err)
	}
	stackPath := "teststack"

	cfgFS := fstest.MapFS{
		// These are somewhat awkward two entries
		// to account for io/fs and our own path separator differences
		// See https://github.com/hashicorp/terraform-ls/issues/1025
		stackPath + "/providers.tfstack.hcl": &fstest.MapFile{
			Data: []byte{},
		},
		filepath.Join(stackPath, "providers.tfstack.hcl"): &fstest.MapFile{
			Data: []byte(`required_providers {
	random = {
		source = "hashicorp/random"
		version = "1.0.0"
	}
	unknown = {
		source = "hashicorp/unknown"
		version = "5.0.0"
	}
}
`),
		},
	}

	err = ss.Add(stackPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx = lsctx.WithDocumentContext(ctx, lsctx.Document{})
	err = ParseStackConfiguration(ctx, cfgFS, ss, stackPath)
	if err != nil {
		t.Fatal(err)
	}
	err = LoadStackMetadata(ctx, ss, stackPath)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, stackPath)
		if err != nil && !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}) {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		err := PreloadEmbeddedSchema(ctx, log.Default(), schemasFS, ss, gs.ProviderSchemas, stackPath)
		if err != nil && !errors.Is(err, job.StateNotChangedErr{Dir: document.DirHandleFromPath(stackPath)}) {
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
