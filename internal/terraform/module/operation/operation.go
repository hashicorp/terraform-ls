// Copyright IBM Corp. 2020, 2026
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
	OpTypeGetInstalledTerraformVersion
	OpTypeObtainSchema
	OpTypeParseModuleConfiguration
	OpTypeParseVariables
	OpTypeParseModuleManifest
	OpTypeParseTerraformSources
	OpTypeLoadModuleMetadata
	OpTypeDecodeReferenceTargets
	OpTypeDecodeReferenceOrigins
	OpTypeDecodeVarsReferences
	OpTypeGetModuleDataFromRegistry
	OpTypeParseProviderVersions
	OpTypePreloadEmbeddedSchema
	OpTypeStacksPreloadEmbeddedSchema
	OpTypeSearchPreloadEmbeddedSchema
	OpTypeSchemaModuleValidation
	OpTypeSchemaStackValidation
	OpTypeSchemaSearchValidation
	OpTypeSchemaVarsValidation
	OpTypeReferenceValidation
	OpTypeReferenceStackValidation
	OpTypeTerraformValidate
	OpTypeParseStackConfiguration
	OpTypeParseSearchConfiguration
	OpTypeParsePolicyConfiguration
	OpTypeLoadPolicyMetadata
	OpTypeSchemaPolicyValidation
	OpTypeReferencePolicyValidation
	OpTypeLoadStackMetadata
	OpTypeLoadSearchMetadata
	OpTypeLoadStackRequiredTerraformVersion
	OpTypeParseTestConfiguration
	OpTypeLoadTestMetadata
	OpTypeDecodeTestReferenceTargets
	OpTypeDecodeTestReferenceOrigins
	OpTypeDecodeWriteOnlyAttributes
	OpTypeSchemaTestValidation
)
