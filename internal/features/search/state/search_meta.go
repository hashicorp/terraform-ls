// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	tfsearch "github.com/hashicorp/terraform-schema/search"
)

// SearchMetadata contains the result of the early decoding of a Search,
// it will be used obtain the correct provider and related module schemas
type SearchMetadata struct {
	CoreRequirements version.Constraints
	Filenames        []string

	Lists     map[string]tfsearch.List
	Variables map[string]tfsearch.Variable

	ProviderReferences   map[tfsearch.ProviderRef]tfaddr.Provider
	ProviderRequirements tfsearch.ProviderRequirements
}

func (sm SearchMetadata) Copy() SearchMetadata {
	newSm := SearchMetadata{
		CoreRequirements: sm.CoreRequirements,
		Filenames:        sm.Filenames,
	}

	if sm.Lists != nil {
		newSm.Lists = make(map[string]tfsearch.List, len(sm.Lists))
		for k, v := range sm.Lists {
			newSm.Lists[k] = v
		}
	}

	if sm.Variables != nil {
		newSm.Variables = make(map[string]tfsearch.Variable, len(sm.Variables))
		for k, v := range sm.Variables {
			newSm.Variables[k] = v
		}
	}

	if sm.ProviderReferences != nil {
		newSm.ProviderReferences = make(map[tfsearch.ProviderRef]tfaddr.Provider, len(sm.ProviderReferences))
		for ref, provider := range sm.ProviderReferences {
			newSm.ProviderReferences[ref] = provider
		}
	}

	if sm.ProviderRequirements != nil {
		newSm.ProviderRequirements = make(tfsearch.ProviderRequirements, len(sm.ProviderRequirements))
		for provider, vc := range sm.ProviderRequirements {
			// version.Constraints is never mutated in this context
			newSm.ProviderRequirements[provider] = vc
		}
	}

	return newSm
}
