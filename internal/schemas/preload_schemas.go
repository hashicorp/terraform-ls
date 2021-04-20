package schemas

import (
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-registry-address"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

func PreloadSchemasToStore(pss *state.ProviderSchemaStore) error {
	pOut, vOut, err := PreloadedProviderSchemas()
	if pOut == nil || err != nil {
		return err
	}

	for rawAddr, pJsonSchema := range pOut.Schemas {
		pv := vOut.Providers[rawAddr]

		pAddr, err := tfaddr.ParseRawProviderSourceString(rawAddr)
		if err != nil {
			// skip unparsable address
			continue
		}
		// Given that we use Terraform >0.12 for the generation
		// this should never happen
		if pAddr.IsLegacy() {
			iAddr, err := tfaddr.ParseAndInferProviderSourceString(rawAddr)
			if err == nil {
				pAddr = iAddr
			}
		}

		pSchema := tfschema.ProviderSchemaFromJson(pJsonSchema, pAddr)
		pSchema.SetProviderVersion(pAddr, pv)
		err = pss.AddPreloadedSchema(pAddr, pv, pSchema)
		if err != nil {
			return err
		}
	}
	return nil
}
