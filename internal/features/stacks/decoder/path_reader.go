// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	stackschema "github.com/hashicorp/terraform-schema/schema"
)

type PathReader struct {
	StateReader StateReader
}

type StateReader interface {
	List() ([]*state.StackRecord, error)
	StackRecordByPath(modPath string) (*state.StackRecord, error)
}

// PathContext returns a PathContext for the given path based on the language ID
func (pr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	record, err := pr.StateReader.StackRecordByPath(path.Path)
	if err != nil {
		return nil, err
	}

	switch path.LanguageID {
	case ilsp.Stacks.String():
		return stackPathContext(record)
	case ilsp.Deploy.String():
		return deployPathContext(record)
	}

	return nil, fmt.Errorf("unknown language ID: %q", path.LanguageID)
}

func stackPathContext(record *state.StackRecord) (*decoder.PathContext, error) {
	// TODO: get Terraform version from record and use that to get the schema
	// TODO: this should only work for terraform 1.8 and above
	schema, err := stackschema.CoreStackSchemaForVersion(stackschema.LatestAvailableVersion)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
	}

	// TODO: Add reference origins and targets if needed

	for name, f := range record.ParsedStackFiles {
		pathCtx.Files[name.String()] = f
	}

	return pathCtx, nil
}

func deployPathContext(record *state.StackRecord) (*decoder.PathContext, error) {
	// TODO: get Terraform version from record and use that to get the schema
	// TODO: this should only work for terraform 1.8 and above
	schema, err := stackschema.CoreDeploySchemaForVersion(stackschema.LatestAvailableVersion)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
	}

	// TODO: Add reference origins and targets if needed

	for name, f := range record.ParsedDeployFiles {
		pathCtx.Files[name.String()] = f
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
		if len(record.ParsedStackFiles) > 0 {
			paths = append(paths, lang.Path{
				Path:       record.Path(),
				LanguageID: ilsp.Stacks.String(),
			})
		}
		if len(record.ParsedDeployFiles) > 0 {
			paths = append(paths, lang.Path{
				Path:       record.Path(),
				LanguageID: ilsp.Deploy.String(),
			})
		}
	}

	return paths
}
