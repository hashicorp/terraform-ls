package decoder

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/state"
	tfmodule "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func schemaForModule(mod *state.Module, schemaReader state.SchemaReader, modReader state.ModuleCallReader) (*schema.BodySchema, error) {
	var coreSchema *schema.BodySchema
	coreRequirements := make(version.Constraints, 0)
	if mod.TerraformVersion != nil {
		var err error
		coreSchema, err = tfschema.CoreModuleSchemaForVersion(mod.TerraformVersion)
		if err != nil {
			return nil, err
		}
		coreRequirements, err = version.NewConstraint(mod.TerraformVersion.String())
		if err != nil {
			return nil, err
		}
	} else {
		coreSchema = tfschema.UniversalCoreModuleSchema()
	}

	sm := tfschema.NewSchemaMerger(coreSchema)
	sm.SetSchemaReader(schemaReader)
	sm.SetTerraformVersion(mod.TerraformVersion)
	sm.SetModuleReader(modReader)

	meta := &tfmodule.Meta{
		Path:                 mod.Path,
		CoreRequirements:     coreRequirements,
		ProviderRequirements: mod.Meta.ProviderRequirements,
		ProviderReferences:   mod.Meta.ProviderReferences,
		Variables:            mod.Meta.Variables,
	}

	return sm.SchemaForModule(meta)
}
