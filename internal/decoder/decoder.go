// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/codelens"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/utm"
	tfmod "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func modulePathContext(mod *state.ModuleRecord, stateReader StateReader) (*decoder.PathContext, error) {
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

	for name, f := range mod.ParsedModuleFiles {
		pathCtx.Files[name.String()] = f
	}

	return pathCtx, nil
}

func varsPathContext(mod *state.VariableRecord, stateReader StateReader) (*decoder.PathContext, error) {
	variables := make(map[string]tfmod.Variable)
	meta, err := stateReader.LocalModuleMeta(mod.Path())
	if err == nil {
		variables = meta.Variables
	}

	schema, err := tfschema.SchemaForVariables(variables, mod.Path())
	if err != nil {
		return nil, err
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File),
	}

	if len(schema.Attributes) > 0 {
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

func DecoderContext(ctx context.Context) decoder.DecoderContext {
	dCtx := decoder.NewDecoderContext()
	dCtx.UtmSource = utm.UtmSource
	dCtx.UtmMedium = utm.UtmMedium(ctx)
	dCtx.UseUtmContent = true

	cc, err := ilsp.ClientCapabilities(ctx)
	if err == nil {
		cmdId, ok := lsp.ExperimentalClientCapabilities(cc.Experimental).ShowReferencesCommandId()
		if ok {
			dCtx.CodeLenses = append(dCtx.CodeLenses, codelens.ReferenceCount(cmdId))
		}
	}

	return dCtx
}
