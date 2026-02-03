// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/variables/ast"
	"github.com/hashicorp/terraform-ls/internal/features/variables/state"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func variablePathContext(mod *state.VariableRecord, moduleReader ModuleReader, useAnySchema bool) (*decoder.PathContext, error) {
	variables, _ := moduleReader.ModuleInputs(mod.Path())
	bodySchema := &schema.BodySchema{}
	if useAnySchema {
		bodySchema = tfschema.AnySchemaForVariableCollection(mod.Path())
	} else {
		var err error
		bodySchema, err = tfschema.SchemaForVariables(variables, mod.Path())
		if err != nil {
			return nil, err
		}
	}

	pathCtx := &decoder.PathContext{
		Schema:           bodySchema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File),
	}

	if len(bodySchema.Attributes) > 0 {
		// Only validate if this is actually a module
		// as we may come across standalone tfvars files
		// for which we have no context.
		pathCtx.Validators = varsValidators
	}

	for _, origin := range mod.VarsRefOrigins {
		if ast.IsVarsFilename(origin.OriginRange().Filename) {
			pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
		}
	}

	for name, f := range mod.ParsedVarsFiles {
		pathCtx.Files[name.String()] = f
	}

	return pathCtx, nil
}
