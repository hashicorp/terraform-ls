package state

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// StackRecord represents a single stack in the state
// /some/path/lambda-multi-account-stack
type StackRecord struct {
	path string

	Meta StackMetadata

	// ParsedStackFiles is the parsed tfstack files for the stack
	// /some/path/lambda-multi-account-stack/components.tfstack.hcl
	ParsedStackFiles ast.StackFiles
	StackParsingErr  error

	StackDiagnostics      ast.SourceStackDiags
	StackDiagnosticsState globalAst.DiagnosticSourceState
}

func (m *StackRecord) Path() string {
	return m.path
}

func (m *StackRecord) Copy() *StackRecord {
	if m == nil {
		return nil
	}

	newRecord := &StackRecord{
		path:                  m.path,
		StackParsingErr:       m.StackParsingErr,
		StackDiagnosticsState: m.StackDiagnosticsState.Copy(),
	}

	if m.ParsedStackFiles != nil {
		newRecord.ParsedStackFiles = make(ast.StackFiles, len(m.ParsedStackFiles))
		for name, f := range m.ParsedStackFiles {
			// hcl.File is practically immutable once it comes out of parser
			newRecord.ParsedStackFiles[name] = f
		}
	}

	if m.StackDiagnostics != nil {
		newRecord.StackDiagnostics = make(ast.SourceStackDiags, len(m.StackDiagnostics))

		for source, stacksDiags := range m.StackDiagnostics {
			newRecord.StackDiagnostics[source] = make(ast.StackDiags, len(stacksDiags))

			for name, diags := range stacksDiags {
				newRecord.StackDiagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newRecord.StackDiagnostics[source][name], diags)
			}
		}
	}

	return newRecord
}

func newStack(modPath string) *StackRecord {
	return &StackRecord{
		path: modPath,
		StackDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          operation.OpStateUnknown,
			globalAst.SchemaValidationSource:    operation.OpStateUnknown,
			globalAst.ReferenceValidationSource: operation.OpStateUnknown,
			globalAst.TerraformValidateSource:   operation.OpStateUnknown,
		},
	}
}
