// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/features/policy/state"
	tfpolicy "github.com/hashicorp/terraform-schema/policy"
	tfschema "github.com/hashicorp/terraform-schema/schema"
	policySchema "github.com/hashicorp/terraform-schema/schema/policy"
)

func schemaForPolicy(policy *state.PolicyRecord, stateReader CombinedReader) (*schema.BodySchema, error) {
	resolvedVersion := tfschema.ResolveVersion(stateReader.TerraformVersion(policy.Path()), policy.Meta.CoreRequirements)
	sm := policySchema.NewSchemaMerger(mustCoreSchemaForVersion(resolvedVersion))
	sm.SetTerraformVersion(resolvedVersion)
	sm.SetStateReader(stateReader)

	meta := &tfpolicy.Meta{
		Path:             policy.Path(),
		CoreRequirements: policy.Meta.CoreRequirements,
		ResourcePolicies: policy.Meta.ResourcePolicies,
		ProviderPolicies: policy.Meta.ProviderPolicies,
		ModulePolicies:   policy.Meta.ModulePolicies,
		Filenames:        policy.Meta.Filenames,
	}

	return sm.SchemaForPolicy(meta)
}

func mustCoreSchemaForVersion(v *version.Version) *schema.BodySchema {
	s, err := policySchema.CorePolicySchemaForVersion(v)
	if err != nil {
		// this should never happen
		panic(err)
	}
	return s
}
