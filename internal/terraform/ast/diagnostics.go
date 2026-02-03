// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package ast

import (
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
	return "Terraform"
}

type DiagnosticSourceState map[DiagnosticSource]op.OpState

func (dss DiagnosticSourceState) Copy() DiagnosticSourceState {
	newDiagnosticSourceState := make(DiagnosticSourceState, len(dss))
	for source, state := range dss {
		newDiagnosticSourceState[source] = state
	}

	return newDiagnosticSourceState
}
