// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/state"
	tfmodule "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func functionsForModule(mod *state.Module, schemaReader state.SchemaReader) (map[string]schema.FunctionSignature, error) {
	resolvedVersion := tfschema.ResolveVersion(mod.TerraformVersion, mod.Meta.CoreRequirements)
	sm := tfschema.NewFunctionsMerger(mustFunctionsForVersion(resolvedVersion))
	sm.SetSchemaReader(schemaReader)

	meta := &tfmodule.Meta{
		Path:                 mod.Path,
		ProviderRequirements: mod.Meta.ProviderRequirements,
		ProviderReferences:   mod.Meta.ProviderReferences,
	}

	return sm.FunctionsForModule(meta)
}

func mustFunctionsForVersion(v *version.Version) map[string]schema.FunctionSignature {
	s, err := tfschema.FunctionsForVersion(v)
	if err != nil {
		// this should never happen
		panic(err)
	}
	return s
}
