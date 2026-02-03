// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform-ls/internal/features/modules/ast"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// ModuleRecord contains all information about module files
// we have for a certain path
type ModuleRecord struct {
	path string

	// PreloadEmbeddedSchemaState tracks if we tried loading all provider
	// schemas from our embedded schema data
	PreloadEmbeddedSchemaState op.OpState

	RefTargets      reference.Targets
	RefTargetsErr   error
	RefTargetsState op.OpState

	RefOrigins      reference.Origins
	RefOriginsErr   error
	RefOriginsState op.OpState

	ParsedModuleFiles ast.ModFiles
	ModuleParsingErr  error

	Meta      ModuleMetadata
	MetaErr   error
	MetaState op.OpState

	WriteOnlyAttributes      WriteOnlyAttributes
	WriteOnlyAttributesErr   error
	WriteOnlyAttributesState op.OpState

	ModuleDiagnostics      ast.SourceModDiags
	ModuleDiagnosticsState globalAst.DiagnosticSourceState
}

func (m *ModuleRecord) Copy() *ModuleRecord {
	if m == nil {
		return nil
	}
	newMod := &ModuleRecord{
		path: m.path,

		PreloadEmbeddedSchemaState: m.PreloadEmbeddedSchemaState,

		RefTargets:      m.RefTargets.Copy(),
		RefTargetsErr:   m.RefTargetsErr,
		RefTargetsState: m.RefTargetsState,

		RefOrigins:      m.RefOrigins.Copy(),
		RefOriginsErr:   m.RefOriginsErr,
		RefOriginsState: m.RefOriginsState,

		ModuleParsingErr: m.ModuleParsingErr,

		Meta:      m.Meta.Copy(),
		MetaErr:   m.MetaErr,
		MetaState: m.MetaState,

		WriteOnlyAttributes:      m.WriteOnlyAttributes,
		WriteOnlyAttributesErr:   m.WriteOnlyAttributesErr,
		WriteOnlyAttributesState: m.WriteOnlyAttributesState,

		ModuleDiagnosticsState: m.ModuleDiagnosticsState.Copy(),
	}

	if m.ParsedModuleFiles != nil {
		newMod.ParsedModuleFiles = make(ast.ModFiles, len(m.ParsedModuleFiles))
		for name, f := range m.ParsedModuleFiles {
			// hcl.File is practically immutable once it comes out of parser
			newMod.ParsedModuleFiles[name] = f
		}
	}

	if m.ModuleDiagnostics != nil {
		newMod.ModuleDiagnostics = make(ast.SourceModDiags, len(m.ModuleDiagnostics))

		for source, modDiags := range m.ModuleDiagnostics {
			newMod.ModuleDiagnostics[source] = make(ast.ModDiags, len(modDiags))

			for name, diags := range modDiags {
				newMod.ModuleDiagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newMod.ModuleDiagnostics[source][name], diags)
			}
		}
	}

	return newMod
}

func (m *ModuleRecord) Path() string {
	return m.path
}

func newModule(modPath string) *ModuleRecord {
	return &ModuleRecord{
		path:                       modPath,
		PreloadEmbeddedSchemaState: op.OpStateUnknown,
		RefOriginsState:            op.OpStateUnknown,
		RefTargetsState:            op.OpStateUnknown,
		MetaState:                  op.OpStateUnknown,
		WriteOnlyAttributesState:   op.OpStateUnknown,
		ModuleDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          op.OpStateUnknown,
			globalAst.SchemaValidationSource:    op.OpStateUnknown,
			globalAst.ReferenceValidationSource: op.OpStateUnknown,
			globalAst.TerraformValidateSource:   op.OpStateUnknown,
		},
	}
}

// NewModuleTest is a test helper to create a new Module object
func NewModuleTest(path string) *ModuleRecord {
	return &ModuleRecord{
		path: path,
	}
}
