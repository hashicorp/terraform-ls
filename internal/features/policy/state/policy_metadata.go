// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/go-version"
	tfpolicy "github.com/hashicorp/terraform-schema/policy"
)

// PolicyMetadata contains the result of the early decoding of a policy,
// it will be used obtain the correct provider and related policy schemas
type PolicyMetadata struct {
	CoreRequirements version.Constraints

	ResourcePolicies map[string]tfpolicy.ResourcePolicy
	ProviderPolicies map[string]tfpolicy.ProviderPolicy
	ModulePolicies   map[string]tfpolicy.ModulePolicy

	Filenames []string
}

func (mm PolicyMetadata) Copy() PolicyMetadata {
	newMm := PolicyMetadata{
		// version.Constraints is practically immutable once parsed
		CoreRequirements: mm.CoreRequirements,
		Filenames:        mm.Filenames,
	}

	if mm.ResourcePolicies != nil {
		newMm.ResourcePolicies = make(map[string]tfpolicy.ResourcePolicy, len(mm.ResourcePolicies))
		for k, v := range mm.ResourcePolicies {
			newMm.ResourcePolicies[k] = v
		}
	}

	if mm.ProviderPolicies != nil {
		newMm.ProviderPolicies = make(map[string]tfpolicy.ProviderPolicy, len(mm.ProviderPolicies))
		for k, v := range mm.ProviderPolicies {
			newMm.ProviderPolicies[k] = v
		}
	}

	if mm.ModulePolicies != nil {
		newMm.ModulePolicies = make(map[string]tfpolicy.ModulePolicy, len(mm.ModulePolicies))
		for k, v := range mm.ModulePolicies {
			newMm.ModulePolicies[k] = v
		}
	}
	return newMm
}
