// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

// TestMetadata contains the result of the early decoding of a test,
// it will be used obtain the correct provider and related module schemas
type TestMetadata struct {
	Filenames []string
}

func (tm TestMetadata) Copy() TestMetadata {
	newTm := TestMetadata{
		Filenames: tm.Filenames,
	}

	return newTm
}
