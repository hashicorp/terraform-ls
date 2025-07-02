// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	tfsearch "github.com/hashicorp/terraform-schema/search"
)

// SearchMetadata contains the result of the early decoding of a Search,
// it will be used obtain the correct provider and related module schemas
type SearchMetadata struct {
	Filenames []string

	Lists     map[string]tfsearch.List
	Variables map[string]tfsearch.Variable
}

func (sm SearchMetadata) Copy() SearchMetadata {
	newSm := SearchMetadata{
		Filenames: sm.Filenames,
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

	return newSm
}
