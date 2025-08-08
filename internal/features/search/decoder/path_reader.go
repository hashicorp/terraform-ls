// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/search/ast"
	"github.com/hashicorp/terraform-ls/internal/features/search/state"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	searchSchema "github.com/hashicorp/terraform-schema/schema/search"
	tfsearch "github.com/hashicorp/terraform-schema/search"
)

type PathReader struct {
	StateReader  StateReader
	ModuleReader ModuleReader
	RootReader   RootReader
}

var _ decoder.PathReader = &PathReader{}

type CombinedReader struct {
	ModuleReader
	StateReader
	RootReader
}

type StateReader interface {
	List() ([]*state.SearchRecord, error)
	SearchRecordByPath(modPath string) (*state.SearchRecord, error)
	ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error)
}

type ModuleReader interface {
	// LocalModuleMeta returns the module meta data for a local module. This is the result
	// of the [earlydecoder] when processing module files
	LocalModuleMeta(modPath string) (*tfmod.Meta, error)
}

type RootReader interface {
	InstalledModulePath(rootPath string, normalizedSource string) (string, bool)

	TerraformVersion(modPath string) *version.Version
}

// PathContext returns a PathContext for the given path based on the language ID
func (pr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	record, err := pr.StateReader.SearchRecordByPath(path.Path)
	if err != nil {
		return nil, err
	}

	switch path.LanguageID {
	case ilsp.Search.String():
		return searchPathContext(record, CombinedReader{
			StateReader:  pr.StateReader,
			ModuleReader: pr.ModuleReader,
			RootReader:   pr.RootReader,
		})
	}

	return nil, fmt.Errorf("unknown language ID: %q", path.LanguageID)
}

func searchPathContext(record *state.SearchRecord, stateReader CombinedReader) (*decoder.PathContext, error) {
	resolvedVersion := tfschema.ResolveVersion(stateReader.TerraformVersion(record.Path()), record.Meta.CoreRequirements)

	sm := searchSchema.NewSearchSchemaMerger(mustCoreSchemaForVersion(resolvedVersion))
	sm.SetStateReader(stateReader)

	meta := &tfsearch.Meta{
		Path:                 record.Path(),
		CoreRequirements:     record.Meta.CoreRequirements,
		Lists:                record.Meta.Lists,
		Variables:            record.Meta.Variables,
		Filenames:            record.Meta.Filenames,
		ProviderReferences:   record.Meta.ProviderReferences,
		ProviderRequirements: record.Meta.ProviderRequirements,
	}

	mergedSchema, err := sm.SchemaForSearch(meta)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           mergedSchema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
		Validators:       searchValidators,
	}

	// TODO: Add reference origins and targets if needed
	for _, origin := range record.RefOrigins {
		if ast.IsSearchFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}

	for _, target := range record.RefTargets {
		if target.RangePtr != nil && ast.IsSearchFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		} else if target.RangePtr == nil {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	for name, f := range record.ParsedFiles {
		if _, ok := name.(ast.SearchFilename); ok {
			pathCtx.Files[name.String()] = f
		}
	}

	return pathCtx, nil
}

func (pr *PathReader) Paths(ctx context.Context) []lang.Path {
	paths := make([]lang.Path, 0)

	searchRecords, err := pr.StateReader.List()
	if err != nil {
		return paths
	}

	for _, record := range searchRecords {
		foundSearch := false
		for name := range record.ParsedFiles {
			if _, ok := name.(ast.SearchFilename); ok {
				foundSearch = true
			}

		}

		if foundSearch {
			paths = append(paths, lang.Path{
				Path:       record.Path(),
				LanguageID: ilsp.Search.String(),
			})
		}

	}

	return paths
}

func mustCoreSchemaForVersion(v *version.Version) *schema.BodySchema {
	s, err := searchSchema.CoreSearchSchemaForVersion(v)
	if err != nil {
		// this should never happen
		panic(err)
	}
	return s
}
