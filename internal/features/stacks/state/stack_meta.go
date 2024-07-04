// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

// StackMetadata contains the result of the early decoding of a module,
// it will be used obtain the correct provider and related module schemas
type StackMetadata struct {
	Filenames []string
}

func (sm StackMetadata) Copy() StackMetadata {
	newSm := StackMetadata{
		// version.Constraints is practically immutable once parsed
		Filenames: sm.Filenames,
	}

	return newSm
}
