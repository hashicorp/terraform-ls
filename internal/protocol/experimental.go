// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package protocol

type ExperimentalServerCapabilities struct {
	ReferenceCountCodeLens  bool `json:"referenceCountCodeLens"`
	RefreshModuleProviders  bool `json:"refreshModuleProviders"`
	RefreshModuleCalls      bool `json:"refreshModuleCalls"`
	RefreshTerraformVersion bool `json:"refreshTerraformVersion"`
}

type ExpClientCapabilities map[string]interface{}

func ExperimentalClientCapabilities(input interface{}) ExpClientCapabilities {
	if m, ok := input.(map[string]interface{}); ok && len(m) > 0 {
		return ExpClientCapabilities(m)
	}
	return make(ExpClientCapabilities, 0)
}

func (cc ExpClientCapabilities) ShowReferencesCommandId() (string, bool) {
	if cc == nil {
		return "", false
	}

	cmdId, ok := cc["showReferencesCommandId"].(string)
	return cmdId, ok
}

func (cc ExpClientCapabilities) RefreshModuleProvidersCommandId() (string, bool) {
	if cc == nil {
		return "", false
	}

	cmdId, ok := cc["refreshModuleProvidersCommandId"].(string)
	return cmdId, ok
}

func (cc ExpClientCapabilities) RefreshModuleCallsCommandId() (string, bool) {
	if cc == nil {
		return "", false
	}

	cmdId, ok := cc["refreshModuleCallsCommandId"].(string)
	return cmdId, ok
}

func (cc ExpClientCapabilities) RefreshTerraformVersionCommandId() (string, bool) {
	if cc == nil {
		return "", false
	}

	cmdId, ok := cc["refreshTerraformVersionCommandId"].(string)
	return cmdId, ok
}

func (cc ExpClientCapabilities) TelemetryVersion() (int, bool) {
	if cc == nil {
		return 0, false
	}

	// numbers are unmarshalled as float64 from JSON
	// per https://pkg.go.dev/encoding/json#Unmarshal
	v, ok := cc["telemetryVersion"].(float64)

	return int(v), ok
}
