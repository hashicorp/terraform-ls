// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform-ls/internal/features/policytest/ast"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// PolicyTestRecord contains all information about policytest files
// we have for a certain path
type PolicyTestRecord struct {
	path string

	RefTargets      reference.Targets
	RefTargetsErr   error
	RefTargetsState op.OpState

	RefOrigins      reference.Origins
	RefOriginsErr   error
	RefOriginsState op.OpState

	ParsedPolicyTestFiles ast.PolicyTestFiles
	PolicyTestParsingErr  error

	Meta      PolicyTestMetadata
	MetaErr   error
	MetaState op.OpState

	PolicyTestDiagnostics      ast.SourcePolicyTestDiags
	PolicyTestDiagnosticsState globalAst.DiagnosticSourceState
}

func (m *PolicyTestRecord) Copy() *PolicyTestRecord {
	if m == nil {
		return nil
	}
	newPolicyTest := &PolicyTestRecord{
		path: m.path,

		RefTargets:      m.RefTargets.Copy(),
		RefTargetsErr:   m.RefTargetsErr,
		RefTargetsState: m.RefTargetsState,

		RefOrigins:      m.RefOrigins.Copy(),
		RefOriginsErr:   m.RefOriginsErr,
		RefOriginsState: m.RefOriginsState,

		PolicyTestParsingErr: m.PolicyTestParsingErr,

		Meta:      m.Meta.Copy(),
		MetaErr:   m.MetaErr,
		MetaState: m.MetaState,

		PolicyTestDiagnosticsState: m.PolicyTestDiagnosticsState.Copy(),
	}

	if m.ParsedPolicyTestFiles != nil {
		newPolicyTest.ParsedPolicyTestFiles = make(ast.PolicyTestFiles, len(m.ParsedPolicyTestFiles))
		for name, f := range m.ParsedPolicyTestFiles {
			// hcl.File is practically immutable once it comes out of parser
			newPolicyTest.ParsedPolicyTestFiles[name] = f
		}
	}

	if m.PolicyTestDiagnostics != nil {
		newPolicyTest.PolicyTestDiagnostics = make(ast.SourcePolicyTestDiags, len(m.PolicyTestDiagnostics))

		for source, policytestDiags := range m.PolicyTestDiagnostics {
			newPolicyTest.PolicyTestDiagnostics[source] = make(ast.PolicyTestDiags, len(policytestDiags))

			for name, diags := range policytestDiags {
				newPolicyTest.PolicyTestDiagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newPolicyTest.PolicyTestDiagnostics[source][name], diags)
			}
		}
	}

	return newPolicyTest
}

func (m *PolicyTestRecord) Path() string {
	return m.path
}

func newPolicyTest(policytestPath string) *PolicyTestRecord {
	return &PolicyTestRecord{
		path:            policytestPath,
		RefOriginsState: op.OpStateUnknown,
		RefTargetsState: op.OpStateUnknown,
		MetaState:       op.OpStateUnknown,
		PolicyTestDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          op.OpStateUnknown,
			globalAst.SchemaValidationSource:    op.OpStateUnknown,
			globalAst.ReferenceValidationSource: op.OpStateUnknown,
			globalAst.TerraformValidateSource:   op.OpStateUnknown,
		},
	}
}

// NewPolicyTestTest is a test helper to create a new PolicyTest object
func NewPolicyTestTest(path string) *PolicyTestRecord {
	return &PolicyTestRecord{
		path: path,
	}
}
