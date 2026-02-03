// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package protocol

const TelemetryFormatVersion = 1

type TelemetryEvent struct {
	Version int `json:"v"`

	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
}
