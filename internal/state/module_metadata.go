// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform-schema/backend"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

// ModuleMetadata contains the result of the early decoding of a module,
// it will be used obtain the correct provider and related module schemas
type ModuleMetadata struct {
	CoreRequirements     version.Constraints
	Backend              *tfmod.Backend
	Cloud                *backend.Cloud
	ProviderReferences   map[tfmod.ProviderRef]tfaddr.Provider
	ProviderRequirements tfmod.ProviderRequirements
	Variables            map[string]tfmod.Variable
	Outputs              map[string]tfmod.Output
	Filenames            []string
	ModuleCalls          map[string]tfmod.DeclaredModuleCall
}

func (mm ModuleMetadata) Copy() ModuleMetadata {
	newMm := ModuleMetadata{
		// version.Constraints is practically immutable once parsed
		CoreRequirements: mm.CoreRequirements,
		Filenames:        mm.Filenames,
	}

	if mm.Cloud != nil {
		newMm.Cloud = mm.Cloud
	}

	if mm.Backend != nil {
		newMm.Backend = &tfmod.Backend{
			Type: mm.Backend.Type,
			Data: mm.Backend.Data.Copy(),
		}
	}

	if mm.ProviderReferences != nil {
		newMm.ProviderReferences = make(map[tfmod.ProviderRef]tfaddr.Provider, len(mm.ProviderReferences))
		for ref, provider := range mm.ProviderReferences {
			newMm.ProviderReferences[ref] = provider
		}
	}

	if mm.ProviderRequirements != nil {
		newMm.ProviderRequirements = make(tfmod.ProviderRequirements, len(mm.ProviderRequirements))
		for provider, vc := range mm.ProviderRequirements {
			// version.Constraints is never mutated in this context
			newMm.ProviderRequirements[provider] = vc
		}
	}

	if mm.Variables != nil {
		newMm.Variables = make(map[string]tfmod.Variable, len(mm.Variables))
		for name, variable := range mm.Variables {
			newMm.Variables[name] = variable
		}
	}

	if mm.Outputs != nil {
		newMm.Outputs = make(map[string]tfmod.Output, len(mm.Outputs))
		for name, output := range mm.Outputs {
			newMm.Outputs[name] = output
		}
	}

	if mm.ModuleCalls != nil {
		newMm.ModuleCalls = make(map[string]tfmod.DeclaredModuleCall, len(mm.ModuleCalls))
		for name, moduleCall := range mm.ModuleCalls {
			newMm.ModuleCalls[name] = moduleCall.Copy()
		}
	}

	return newMm
}
