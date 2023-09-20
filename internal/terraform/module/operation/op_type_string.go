// Code generated by "stringer -type=OpType -output=op_type_string.go"; DO NOT EDIT.

package operation

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[OpTypeUnknown-0]
	_ = x[OpTypeGetTerraformVersion-1]
	_ = x[OpTypeObtainSchema-2]
	_ = x[OpTypeParseModuleConfiguration-3]
	_ = x[OpTypeParseVariables-4]
	_ = x[OpTypeParseModuleManifest-5]
	_ = x[OpTypeLoadModuleMetadata-6]
	_ = x[OpTypeDecodeReferenceTargets-7]
	_ = x[OpTypeDecodeReferenceOrigins-8]
	_ = x[OpTypeDecodeVarsReferences-9]
	_ = x[OpTypeGetModuleDataFromRegistry-10]
	_ = x[OpTypeParseProviderVersions-11]
	_ = x[OpTypePreloadEmbeddedSchema-12]
	_ = x[OpTypeSchemaModuleValidation-13]
	_ = x[OpTypeSchemaVarsValidation-14]
	_ = x[OpTypeReferenceValidation-15]
	_ = x[OpTypeTerraformValidate-16]
}

const _OpType_name = "OpTypeUnknownOpTypeGetTerraformVersionOpTypeObtainSchemaOpTypeParseModuleConfigurationOpTypeParseVariablesOpTypeParseModuleManifestOpTypeLoadModuleMetadataOpTypeDecodeReferenceTargetsOpTypeDecodeReferenceOriginsOpTypeDecodeVarsReferencesOpTypeGetModuleDataFromRegistryOpTypeParseProviderVersionsOpTypePreloadEmbeddedSchemaOpTypeSchemaModuleValidationOpTypeSchemaVarsValidationOpTypeReferenceValidationOpTypeTerraformValidate"

var _OpType_index = [...]uint16{0, 13, 38, 56, 86, 106, 131, 155, 183, 211, 237, 268, 295, 322, 350, 376, 401, 424}

func (i OpType) String() string {
	if i >= OpType(len(_OpType_index)-1) {
		return "OpType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _OpType_name[_OpType_index[i]:_OpType_index[i+1]]
}
