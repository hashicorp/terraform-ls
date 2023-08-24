// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ast

import "fmt"

// DiagnosticSource differentiates different sources of diagnostics.
type DiagnosticSource int

const (
	ModuleParsingSource DiagnosticSource = iota
	VarsParsingSource
	SchemaValidationSource
	ReferenceValidationSource
	TerraformValidateSource
)

func (d DiagnosticSource) String() string {
	switch d {
	case ModuleParsingSource:
		return "HCL"
	case VarsParsingSource:
		return "HCL Vars"
	case SchemaValidationSource:
		return "schema validation"
	case ReferenceValidationSource:
		return "reference validation"
	case TerraformValidateSource:
		return "terraform validate"
	default:
		return fmt.Sprintf("Unknown %d", d)
	}
}

// TODO? combine with langserver/diagnostics
