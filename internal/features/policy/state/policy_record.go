// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform-ls/internal/features/policy/ast"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// PolicyRecord contains all information about policy files
// we have for a certain path
type PolicyRecord struct {
	path string

	RefTargets      reference.Targets
	RefTargetsErr   error
	RefTargetsState op.OpState

	RefOrigins      reference.Origins
	RefOriginsErr   error
	RefOriginsState op.OpState

	ParsedPolicyFiles ast.PolicyFiles
	PolicyParsingErr  error

	Meta      PolicyMetadata
	MetaErr   error
	MetaState op.OpState

	PolicyDiagnostics      ast.SourcePolicyDiags
	PolicyDiagnosticsState globalAst.DiagnosticSourceState
}

func (m *PolicyRecord) Copy() *PolicyRecord {
	if m == nil {
		return nil
	}
	newPolicy := &PolicyRecord{
		path: m.path,

		RefTargets:      m.RefTargets.Copy(),
		RefTargetsErr:   m.RefTargetsErr,
		RefTargetsState: m.RefTargetsState,

		RefOrigins:      m.RefOrigins.Copy(),
		RefOriginsErr:   m.RefOriginsErr,
		RefOriginsState: m.RefOriginsState,

		PolicyParsingErr: m.PolicyParsingErr,

		Meta:      m.Meta.Copy(),
		MetaErr:   m.MetaErr,
		MetaState: m.MetaState,

		PolicyDiagnosticsState: m.PolicyDiagnosticsState.Copy(),
	}

	if m.ParsedPolicyFiles != nil {
		newPolicy.ParsedPolicyFiles = make(ast.PolicyFiles, len(m.ParsedPolicyFiles))
		for name, f := range m.ParsedPolicyFiles {
			// hcl.File is practically immutable once it comes out of parser
			newPolicy.ParsedPolicyFiles[name] = f
		}
	}

	if m.PolicyDiagnostics != nil {
		newPolicy.PolicyDiagnostics = make(ast.SourcePolicyDiags, len(m.PolicyDiagnostics))

		for source, policyDiags := range m.PolicyDiagnostics {
			newPolicy.PolicyDiagnostics[source] = make(ast.PolicyDiags, len(policyDiags))

			for name, diags := range policyDiags {
				newPolicy.PolicyDiagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newPolicy.PolicyDiagnostics[source][name], diags)
			}
		}
	}

	return newPolicy
}

func (m *PolicyRecord) Path() string {
	return m.path
}

func newPolicy(policyPath string) *PolicyRecord {
	return &PolicyRecord{
		path:            policyPath,
		RefOriginsState: op.OpStateUnknown,
		RefTargetsState: op.OpStateUnknown,
		MetaState:       op.OpStateUnknown,
		PolicyDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          op.OpStateUnknown,
			globalAst.SchemaValidationSource:    op.OpStateUnknown,
			globalAst.ReferenceValidationSource: op.OpStateUnknown,
			globalAst.TerraformValidateSource:   op.OpStateUnknown,
		},
	}
}

// NewPolicyTest is a test helper to create a new Policy object
func NewPolicyTest(path string) *PolicyRecord {
	return &PolicyRecord{
		path: path,
	}
}
