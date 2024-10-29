// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/modules/ast"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
)

func modulePathContext(mod *state.ModuleRecord, stateReader CombinedReader) (*decoder.PathContext, error) {
	schema, err := schemaForModule(mod, stateReader)
	if err != nil {
		return nil, err
	}
	functions, err := functionsForModule(mod, stateReader)
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
		Functions:        functions,
		Validators:       moduleValidators,
	}

	for _, origin := range mod.RefOrigins {
		if ast.IsModuleFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}
	for _, target := range mod.RefTargets {
		if target.RangePtr != nil && ast.IsModuleFilename(target.RangePtr.Filename) {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		} else if target.RangePtr == nil {
			pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
		}
	}

	// append Terraform version specific path targets that are available in all modules (builtin references)
	pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, referencesForModule(mod, stateReader)...)

	for name, f := range mod.ParsedModuleFiles {
		pathCtx.Files[name.String()] = f
	}

	return pathCtx, nil
}
