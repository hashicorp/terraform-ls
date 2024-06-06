// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/features/modules/state"
	tfmodule "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func schemaForModule(mod *state.ModuleRecord, stateReader CombinedReader) (*schema.BodySchema, error) {
	resolvedVersion := tfschema.ResolveVersion(stateReader.TerraformVersion(mod.Path()), mod.Meta.CoreRequirements)
	sm := tfschema.NewSchemaMerger(mustCoreSchemaForVersion(resolvedVersion))
	sm.SetTerraformVersion(resolvedVersion)
	sm.SetStateReader(stateReader)

	meta := &tfmodule.Meta{
		Path:                 mod.Path(),
		CoreRequirements:     mod.Meta.CoreRequirements,
		ProviderRequirements: mod.Meta.ProviderRequirements,
		ProviderReferences:   mod.Meta.ProviderReferences,
		Variables:            mod.Meta.Variables,
		Filenames:            mod.Meta.Filenames,
		ModuleCalls:          mod.Meta.ModuleCalls,
	}

	return sm.SchemaForModule(meta)
}

func mustCoreSchemaForVersion(v *version.Version) *schema.BodySchema {
	s, err := tfschema.CoreModuleSchemaForVersion(v)
	if err != nil {
		// this should never happen
		panic(err)
	}
	return s
}
