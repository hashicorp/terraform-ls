// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

// PolicyTestMetadata contains the result of the early decoding of a policytest,
// it will be used obtain the correct provider and related policytest schemas
type PolicyTestMetadata struct {
	Filenames []string
}

func (mm PolicyTestMetadata) Copy() PolicyTestMetadata {
	newMm := PolicyTestMetadata{
		Filenames: mm.Filenames,
	}

	return newMm
}
