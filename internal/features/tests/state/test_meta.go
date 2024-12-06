// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

// TestMetadata contains the result of the early decoding of a test,
// it will be used obtain the correct provider and related module schemas
type TestMetadata struct {

	// ProviderRequirements map[string]ProviderRequirement
	// MockProviders        map[string]MockProvider

	// TODO: need to collect run blocks and use their module source for kicking of decoding for terraform files on those dirs
	// TODO: testrecord needs to be file not directory based (because the are not merged) (this might be hard)
	// alternatively: another layer for metadata (key them by filename)

	RunBlocks []string // TODO: just for testing, change to proper type

	// Components           map[string]Component
	// Variables            map[string]Variable
	// Outputs              map[string]Output
}

func (tm TestMetadata) Copy() TestMetadata {
	newTm := TestMetadata{}

	return newTm
}
