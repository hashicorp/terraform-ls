// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/tests/ast"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// TestRecord represents a test location in the state
type TestRecord struct {
	path string

	PreloadEmbeddedSchemaState op.OpState

	Meta      TestMetadata
	MetaErr   error
	MetaState op.OpState

	RefTargets      reference.Targets
	RefTargetsErr   error
	RefTargetsState op.OpState

	RefOrigins      reference.Origins
	RefOriginsErr   error
	RefOriginsState op.OpState

	ParsedFiles      ast.Files
	ParsingErr       error
	Diagnostics      ast.SourceDiagnostics
	DiagnosticsState globalAst.DiagnosticSourceState
}

func (m *TestRecord) Path() string {
	return m.path
}

func (m *TestRecord) Copy() *TestRecord {
	if m == nil {
		return nil
	}

	newRecord := &TestRecord{
		path: m.path,

		PreloadEmbeddedSchemaState: m.PreloadEmbeddedSchemaState,

		RefTargets:      m.RefTargets.Copy(),
		RefTargetsErr:   m.RefTargetsErr,
		RefTargetsState: m.RefTargetsState,

		RefOrigins:      m.RefOrigins.Copy(),
		RefOriginsErr:   m.RefOriginsErr,
		RefOriginsState: m.RefOriginsState,

		Meta:      m.Meta.Copy(),
		MetaErr:   m.MetaErr,
		MetaState: m.MetaState,

		ParsingErr:       m.ParsingErr,
		DiagnosticsState: m.DiagnosticsState.Copy(),
	}

	if m.ParsedFiles != nil {
		newRecord.ParsedFiles = make(ast.Files, len(m.ParsedFiles))
		for name, f := range m.ParsedFiles {
			// hcl.File is practically immutable once it comes out of parser
			newRecord.ParsedFiles[name] = f
		}
	}

	if m.Diagnostics != nil {
		newRecord.Diagnostics = make(ast.SourceDiagnostics, len(m.Diagnostics))

		for source, testDiags := range m.Diagnostics {
			newRecord.Diagnostics[source] = make(ast.Diagnostics, len(testDiags))

			for name, diags := range testDiags {
				newRecord.Diagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newRecord.Diagnostics[source][name], diags)
			}
		}
	}

	return newRecord
}

func newTest(testPath string) *TestRecord {
	return &TestRecord{
		path:                       testPath,
		PreloadEmbeddedSchemaState: op.OpStateUnknown,
		RefOriginsState:            op.OpStateUnknown,
		RefTargetsState:            op.OpStateUnknown,
		MetaState:                  op.OpStateUnknown,
		DiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          op.OpStateUnknown,
			globalAst.SchemaValidationSource:    op.OpStateUnknown,
			globalAst.ReferenceValidationSource: op.OpStateUnknown,
			globalAst.TerraformValidateSource:   op.OpStateUnknown,
		},
	}
}
