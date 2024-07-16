// Copyright (c) HashiCorp, Inc.
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
}

func (sm StackMetadata) Copy() StackMetadata {
	newSm := StackMetadata{
		Filenames:            sm.Filenames,
		Components:           sm.Components,
		Variables:            sm.Variables,
		Outputs:              sm.Outputs,
		ProviderRequirements: sm.ProviderRequirements,
	}

	return newSm
}
