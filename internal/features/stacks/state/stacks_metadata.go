// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/go-version"
)

// StackMetadata contains the result of the early decoding of a module,
// it will be used obtain the correct provider and related module schemas
type StackMetadata struct {
	CoreRequirements version.Constraints
	Filenames        []string
}
