// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ast

import (
	"fmt"

	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// DiagnosticSource differentiates different sources of diagnostics.
type DiagnosticSource int

const (
	HCLParsingSource DiagnosticSource = iota
	SchemaValidationSource
	ReferenceValidationSource
	TerraformValidateSource
)

func (d DiagnosticSource) String() string {
	switch d {
	case HCLParsingSource:
		return "HCL"
	case SchemaValidationSource:
		return "early validation"
	case ReferenceValidationSource:
		return "early validation"
	case TerraformValidateSource:
		return "terraform validate"
	default:
		panic(fmt.Sprintf("Unknown diagnostic source %d", d))
	}
}

type DiagnosticSourceState map[DiagnosticSource]op.OpState

func (dss DiagnosticSourceState) Copy() DiagnosticSourceState {
	newDiagnosticSourceState := make(DiagnosticSourceState, len(dss))
	for source, state := range dss {
		newDiagnosticSourceState[source] = state
	}

	return newDiagnosticSourceState
}
