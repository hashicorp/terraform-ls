// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/variables/ast"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// VariableRecord contains all information about variable definition files
// we have for a certain path
type VariableRecord struct {
	path string

	VarsRefOrigins      reference.Origins
	VarsRefOriginsErr   error
	VarsRefOriginsState op.OpState

	ParsedVarsFiles ast.VarsFiles
	VarsParsingErr  error

	VarsDiagnostics      ast.SourceVarsDiags
	VarsDiagnosticsState globalAst.DiagnosticSourceState
}

func (v *VariableRecord) Copy() *VariableRecord {
	if v == nil {
		return nil
	}
	newMod := &VariableRecord{
		path: v.path,

		VarsRefOrigins:      v.VarsRefOrigins.Copy(),
		VarsRefOriginsErr:   v.VarsRefOriginsErr,
		VarsRefOriginsState: v.VarsRefOriginsState,

		VarsParsingErr: v.VarsParsingErr,

		VarsDiagnosticsState: v.VarsDiagnosticsState.Copy(),
	}

	if v.ParsedVarsFiles != nil {
		newMod.ParsedVarsFiles = make(ast.VarsFiles, len(v.ParsedVarsFiles))
		for name, f := range v.ParsedVarsFiles {
			// hcl.File is practically immutable once it comes out of parser
			newMod.ParsedVarsFiles[name] = f
		}
	}

	if v.VarsDiagnostics != nil {
		newMod.VarsDiagnostics = make(ast.SourceVarsDiags, len(v.VarsDiagnostics))

		for source, varsDiags := range v.VarsDiagnostics {
			newMod.VarsDiagnostics[source] = make(ast.VarsDiags, len(varsDiags))

			for name, diags := range varsDiags {
				newMod.VarsDiagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newMod.VarsDiagnostics[source][name], diags)
			}
		}
	}

	return newMod
}

func (v *VariableRecord) Path() string {
	return v.path
}

func newVariableRecord(modPath string) *VariableRecord {
	return &VariableRecord{
		path: modPath,
		VarsDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          op.OpStateUnknown,
			globalAst.SchemaValidationSource:    op.OpStateUnknown,
			globalAst.ReferenceValidationSource: op.OpStateUnknown,
			globalAst.TerraformValidateSource:   op.OpStateUnknown,
		},
	}
}

// NewVariableRecordTest is a test helper to create a new VariableRecord
func NewVariableRecordTest(path string) *VariableRecord {
	return &VariableRecord{
		path: path,
	}
}
