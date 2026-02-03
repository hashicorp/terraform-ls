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
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfmod "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	stackschema "github.com/hashicorp/terraform-schema/schema/stacks"
	tfstack "github.com/hashicorp/terraform-schema/stack"
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
	List() ([]*state.StackRecord, error)
	StackRecordByPath(modPath string) (*state.StackRecord, error)
	ProviderSchema(modPath string, addr tfaddr.Provider, vc version.Constraints) (*tfschema.ProviderSchema, error)
}

type ModuleReader interface {
	// LocalModuleMeta returns the module meta data for a local module. This is the result
	// of the [earlydecoder] when processing module files
	LocalModuleMeta(modPath string) (*tfmod.Meta, error)
}

type RootReader interface {
	InstalledModulePath(rootPath string, normalizedSource string) (string, bool)
}

// PathContext returns a PathContext for the given path based on the language ID
func (pr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	record, err := pr.StateReader.StackRecordByPath(path.Path)
	if err != nil {
		return nil, err
	}

	switch path.LanguageID {
	case ilsp.Stacks.String():
		return stackPathContext(record, CombinedReader{
			StateReader:  pr.StateReader,
			ModuleReader: pr.ModuleReader,
			RootReader:   pr.RootReader,
		})
	case ilsp.Deploy.String():
		return deployPathContext(record)
	}

	return nil, fmt.Errorf("unknown language ID: %q", path.LanguageID)
}

func stackPathContext(record *state.StackRecord, stateReader CombinedReader) (*decoder.PathContext, error) {
	// TODO: this should only work for terraform 1.8 and above
	version := record.RequiredTerraformVersion
	if version == nil {
		version = tfschema.LatestAvailableVersion
	}

	schema, err := stackschema.CoreStackSchemaForVersion(version)
	if err != nil {
		return nil, err
	}

	sm := stackschema.NewStackSchemaMerger(schema)
	sm.SetStateReader(stateReader)

	meta := &tfstack.Meta{
		Path:                 record.Path(),
		ProviderRequirements: record.Meta.ProviderRequirements,
		Components:           record.Meta.Components,
		Variables:            record.Meta.Variables,
		Outputs:              record.Meta.Outputs,
		Filenames:            record.Meta.Filenames,
	}

	mergedSchema, err := sm.SchemaForStack(meta)
	if err != nil {
		return nil, err
	}

	functions, err := functionsForStack(record, version, stateReader)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           mergedSchema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
		Functions:        functions,
		Validators:       stackValidators,
	}

	// TODO: Add reference origins and targets if needed
	for _, origin := range record.RefOrigins {
		if ast.IsStackFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}

	for _, target := range record.RefTargets {
		if target.RangePtr != nil && ast.IsStackFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		} else if target.RangePtr == nil {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	for name, f := range record.ParsedFiles {
		if _, ok := name.(ast.StackFilename); ok {
			pathCtx.Files[name.String()] = f
		}
	}

	return pathCtx, nil
}

func deployPathContext(record *state.StackRecord) (*decoder.PathContext, error) {
	// TODO: this should only work for terraform 1.8 and above
	version := record.RequiredTerraformVersion
	if version == nil {
		version = tfschema.LatestAvailableVersion
	}

	schema, err := stackschema.CoreDeploySchemaForVersion(version)
	if err != nil {
		return nil, err
	}

	sm := stackschema.NewDeploySchemaMerger(schema)

	meta := &tfstack.Meta{
		Path:                 record.Path(),
		ProviderRequirements: record.Meta.ProviderRequirements,
		Components:           record.Meta.Components,
		Variables:            record.Meta.Variables,
		Outputs:              record.Meta.Outputs,
		Filenames:            record.Meta.Filenames,
	}

	mergedSchema, err := sm.SchemaForDeployment(meta)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           mergedSchema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
		Validators:       stackValidators,
		Functions:        deployFunctionsForVersion(version),
	}

	// TODO: Add reference origins and targets if needed
	for _, origin := range record.RefOrigins {
		if ast.IsDeployFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}

	for _, target := range record.RefTargets {
		if target.RangePtr != nil && ast.IsDeployFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		} else if target.RangePtr == nil {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	for name, f := range record.ParsedFiles {
		if _, ok := name.(ast.DeployFilename); ok {
			pathCtx.Files[name.String()] = f
		}
	}

	return pathCtx, nil
}

func (pr *PathReader) Paths(ctx context.Context) []lang.Path {
	paths := make([]lang.Path, 0)

	stackRecords, err := pr.StateReader.List()
	if err != nil {
		return paths
	}

	for _, record := range stackRecords {
		foundStack := false
		foundDeploy := false
		for name := range record.ParsedFiles {
			if _, ok := name.(ast.StackFilename); ok {
				foundStack = true
			}
			if _, ok := name.(ast.DeployFilename); ok {
				foundDeploy = true
			}
		}

		if foundStack {
			paths = append(paths, lang.Path{
				Path:       record.Path(),
				LanguageID: ilsp.Stacks.String(),
			})
		}
		if foundDeploy {
			paths = append(paths, lang.Path{
				Path:       record.Path(),
				LanguageID: ilsp.Deploy.String(),
			})
		}
	}

	return paths
}

func mustFunctionsForVersion(v *version.Version) map[string]schema.FunctionSignature {
	s, err := tfschema.FunctionsForVersion(v)
	if err != nil {
		// this should never happen
		panic(err)
	}
	return s
}
