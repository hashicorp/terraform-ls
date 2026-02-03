// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/tests/ast"
	"github.com/hashicorp/terraform-ls/internal/features/tests/state"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	testschema "github.com/hashicorp/terraform-schema/schema/tests"
	tftest "github.com/hashicorp/terraform-schema/test"
)

type PathReader struct {
	StateReader  StateReader
	ModuleReader ModuleReader
	RootReader   RootReader
}

type StateReader interface {
	List() ([]*state.TestRecord, error)
	TestRecordByPath(modPath string) (*state.TestRecord, error)
	ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error)
}

type ModuleReader interface {
	LocalModuleMeta(modPath string) (*tfmod.Meta, error)
}

type RootReader interface {
	TerraformVersion(modPath string) *version.Version
}

type CombinedReader struct {
	ModuleReader
	StateReader
	RootReader
}

var _ decoder.PathReader = &PathReader{}

// PathContext returns a PathContext for the given path based on the language ID
func (pr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	record, err := pr.StateReader.TestRecordByPath(path.Path)
	if err != nil {
		return nil, err
	}

	switch path.LanguageID {
	case ilsp.Test.String():
		return testPathContext(record, CombinedReader{
			StateReader:  pr.StateReader,
			ModuleReader: pr.ModuleReader,
			RootReader:   pr.RootReader,
		})
	case ilsp.Mock.String():
		return mockPathContext(record, CombinedReader{
			StateReader:  pr.StateReader,
			ModuleReader: pr.ModuleReader,
			RootReader:   pr.RootReader,
		})
	}

	return nil, fmt.Errorf("unknown language ID: %q", path.LanguageID)
}

func testPathContext(record *state.TestRecord, stateReader CombinedReader) (*decoder.PathContext, error) {
	// TODO! this should only work for terraform 1.6 and above
	version := stateReader.TerraformVersion(record.Path())
	if version == nil {
		version = tfschema.LatestAvailableVersion
	}

	schema, err := testschema.CoreTestSchemaForVersion(version)
	if err != nil {
		return nil, err
	}

	sm := testschema.NewTestSchemaMerger(schema)
	sm.SetStateReader(stateReader)

	meta := &tftest.Meta{
		Path:      record.Path(),
		Filenames: record.Meta.Filenames,
	}

	mergedSchema, err := sm.SchemaForTest(meta)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           mergedSchema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
		Validators:       validators,
		// TODO? functions TFECO-7480
	}

	for _, origin := range record.RefOrigins {
		if ast.IsTestFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}
	for _, target := range record.RefTargets {
		if target.RangePtr != nil && ast.IsTestFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	for name, f := range record.ParsedFiles {
		if _, ok := name.(ast.TestFilename); ok {
			pathCtx.Files[name.String()] = f
		}
	}

	return pathCtx, nil
}

func mockPathContext(record *state.TestRecord, stateReader CombinedReader) (*decoder.PathContext, error) {
	// TODO! this should only work for terraform 1.7 and above
	version := stateReader.TerraformVersion(record.Path())
	if version == nil {
		version = tfschema.LatestAvailableVersion
	}

	schema, err := testschema.CoreMockSchemaForVersion(version)
	if err != nil {
		return nil, err
	}

	sm := testschema.NewMockSchemaMerger(schema)
	sm.SetStateReader(stateReader)

	meta := &tftest.Meta{
		Path:      record.Path(),
		Filenames: record.Meta.Filenames,
	}

	mergedSchema, err := sm.SchemaForMock(meta)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           mergedSchema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
		Validators:       validators,
		// TODO? functions TFECO-7480
	}

	for _, origin := range record.RefOrigins {
		if ast.IsMockFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}
	for _, target := range record.RefTargets {
		if target.RangePtr != nil && ast.IsMockFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	for name, f := range record.ParsedFiles {
		if _, ok := name.(ast.MockFilename); ok {
			pathCtx.Files[name.String()] = f
		}
	}

	return pathCtx, nil
}

func (pr *PathReader) Paths(ctx context.Context) []lang.Path {
	paths := make([]lang.Path, 0)

	testRecords, err := pr.StateReader.List()
	if err != nil {
		return paths
	}

	for _, record := range testRecords {
		foundTest := false
		foundMock := false
		for name := range record.ParsedFiles {
			if _, ok := name.(ast.TestFilename); ok {
				foundTest = true
			}
			if _, ok := name.(ast.MockFilename); ok {
				foundMock = true
			}
		}

		if foundTest {
			paths = append(paths, lang.Path{
				Path:       record.Path(),
				LanguageID: ilsp.Test.String(),
			})
		}
		if foundMock {
			paths = append(paths, lang.Path{
				Path:       record.Path(),
				LanguageID: ilsp.Mock.String(),
			})
		}
	}

	return paths
}
