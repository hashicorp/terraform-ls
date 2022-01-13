package operation

//go:generate stringer -type=OpState -output=op_state_string.go
type OpState uint

const (
	OpStateUnknown OpState = iota
	OpStateQueued
	OpStateLoading
	OpStateLoaded
)

//go:generate stringer -type=OpType -output=op_type_string.go
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
)
