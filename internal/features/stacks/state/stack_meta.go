// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	tfstack "github.com/hashicorp/terraform-schema/stack"
)

// StackMetadata contains the result of the early decoding of a Stack,
// it will be used obtain the correct provider and related module schemas
type StackMetadata struct {
	Filenames            []string
	Components           map[string]tfstack.Component
	Variables            map[string]tfstack.Variable
	Outputs              map[string]tfstack.Output
	ProviderRequirements map[string]tfstack.ProviderRequirement

	Deployments        map[string]tfstack.Deployment
	Stores             map[string]tfstack.Store
	OrchestrationRules map[string]tfstack.OrchestrationRule
}

func (sm StackMetadata) Copy() StackMetadata {
	newSm := StackMetadata{
		Filenames: sm.Filenames,
	}

	if sm.Components != nil {
		newSm.Components = make(map[string]tfstack.Component, len(sm.Components))
		for k, v := range sm.Components {
			newSm.Components[k] = v
		}
	}

	if sm.Variables != nil {
		newSm.Variables = make(map[string]tfstack.Variable, len(sm.Variables))
		for k, v := range sm.Variables {
			newSm.Variables[k] = v
		}
	}

	if sm.Outputs != nil {
		newSm.Outputs = make(map[string]tfstack.Output, len(sm.Outputs))
		for k, v := range sm.Outputs {
			newSm.Outputs[k] = v
		}
	}

	if sm.ProviderRequirements != nil {
		newSm.ProviderRequirements = make(map[string]tfstack.ProviderRequirement, len(sm.ProviderRequirements))
		for k, v := range sm.ProviderRequirements {
			newSm.ProviderRequirements[k] = v
		}
	}

	if sm.Deployments != nil {
		newSm.Deployments = make(map[string]tfstack.Deployment, len(sm.Deployments))
		for k, v := range sm.Deployments {
			newSm.Deployments[k] = v
		}
	}

	if sm.Stores != nil {
		newSm.Stores = make(map[string]tfstack.Store, len(sm.Stores))
		for k, v := range sm.Stores {
			newSm.Stores[k] = v
		}
	}

	if sm.OrchestrationRules != nil {
		newSm.OrchestrationRules = make(map[string]tfstack.OrchestrationRule, len(sm.OrchestrationRules))
		for k, v := range sm.OrchestrationRules {
			newSm.OrchestrationRules[k] = v
		}
	}

	return newSm
}
