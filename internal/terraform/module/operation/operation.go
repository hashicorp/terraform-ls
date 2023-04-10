// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package operation

//go:generate go run golang.org/x/tools/cmd/stringer -type=OpState -output=op_state_string.go
type OpState uint

const (
	OpStateUnknown OpState = iota
	OpStateQueued
	OpStateLoading
	OpStateLoaded
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=OpType -output=op_type_string.go
type OpType uint

const (
	OpTypeUnknown OpType = iota
	OpTypeGetTerraformVersion
	OpTypeObtainSchema
	OpTypeParseModuleConfiguration
	OpTypeParseVariables
	OpTypeParseModuleManifest
	OpTypeLoadModuleMetadata
	OpTypeDecodeReferenceTargets
	OpTypeDecodeReferenceOrigins
	OpTypeDecodeVarsReferences
	OpTypeGetModuleDataFromRegistry
	OpTypeParseProviderVersions
	OpTypePreloadEmbeddedSchema
)
