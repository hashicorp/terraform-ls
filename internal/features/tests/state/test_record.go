// Copyright (c) HashiCorp, Inc.
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

	// Mapping of filename to Metadata (same for the other maps below)
	Meta      map[string]TestMetadata
	MetaErr   error
	MetaState op.OpState

	GlobalRefTargets reference.Targets // these are global targets that are not tied to a specific file
	RefTargets       map[string]reference.Targets
	RefTargetsErr    error
	RefTargetsState  op.OpState

	GlobalRefOrigins reference.Origins // these are global origins that are not tied to a specific file
	RefOrigins       map[string]reference.Origins
	RefOriginsErr    error
	RefOriginsState  op.OpState

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

		RefTargetsErr:   m.RefTargetsErr,
		RefTargetsState: m.RefTargetsState,

		RefOriginsErr:   m.RefOriginsErr,
		RefOriginsState: m.RefOriginsState,

		MetaErr:   m.MetaErr,
		MetaState: m.MetaState,

		ParsingErr:       m.ParsingErr,
		DiagnosticsState: m.DiagnosticsState.Copy(),
	}

	if m.RefTargets != nil {
		newRecord.RefTargets = make(map[string]reference.Targets, len(m.RefTargets))
		for name, targets := range m.RefTargets {
			newRecord.RefTargets[name] = targets.Copy()
		}
	}

	if m.RefOrigins != nil {
		newRecord.RefOrigins = make(map[string]reference.Origins, len(m.RefOrigins))
		for name, origins := range m.RefOrigins {
			newRecord.RefOrigins[name] = origins.Copy()
		}
	}

	if m.Meta != nil {
		newRecord.Meta = make(map[string]TestMetadata, len(m.Meta))
		for name, meta := range m.Meta {
			newRecord.Meta[name] = meta.Copy()
		}
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
