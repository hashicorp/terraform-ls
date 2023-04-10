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

func schemaForModule(mod *state.Module, schemaReader state.SchemaReader, modReader state.ModuleCallReader) (*schema.BodySchema, error) {
	sm := tfschema.NewSchemaMerger(coreSchema(mod))
	sm.SetSchemaReader(schemaReader)
	sm.SetTerraformVersion(mod.TerraformVersion)
	sm.SetModuleReader(modReader)

	meta := &tfmodule.Meta{
		Path:                 mod.Path,
		CoreRequirements:     mod.Meta.CoreRequirements,
		ProviderRequirements: mod.Meta.ProviderRequirements,
		ProviderReferences:   mod.Meta.ProviderReferences,
		Variables:            mod.Meta.Variables,
		Filenames:            mod.Meta.Filenames,
		ModuleCalls:          mod.Meta.ModuleCalls,
	}

	return sm.SchemaForModule(meta)
}

func coreSchema(mod *state.Module) *schema.BodySchema {
	if mod.TerraformVersion != nil {
		s, err := tfschema.CoreModuleSchemaForVersion(mod.TerraformVersion)
		if err == nil {
			return s
		}
		if mod.TerraformVersion.LessThan(tfschema.OldestAvailableVersion) {
			return mustCoreSchemaForVersion(tfschema.OldestAvailableVersion)
		}

		return mustCoreSchemaForVersion(tfschema.LatestAvailableVersion)
	}

	s, err := tfschema.CoreModuleSchemaForConstraint(mod.Meta.CoreRequirements)
	if err == nil {
		return s
	}
	if mod.Meta.CoreRequirements.Check(tfschema.OldestAvailableVersion) {
		return mustCoreSchemaForVersion(tfschema.OldestAvailableVersion)
	}

	return mustCoreSchemaForVersion(tfschema.LatestAvailableVersion)
}

func mustCoreSchemaForVersion(v *version.Version) *schema.BodySchema {
	s, err := tfschema.CoreModuleSchemaForVersion(v)
	if err != nil {
		// this should never happen
		panic(err)
	}
	return s
}
