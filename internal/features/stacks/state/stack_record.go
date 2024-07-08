// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/go-version"
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

	// ParsedFiles is a map of all the parsed files for the stack,
	// including Stack and Deploy files.
	ParsedFiles      ast.Files
	ParsingErr       error
	Diagnostics      ast.SourceDiagnostics
	DiagnosticsState globalAst.DiagnosticSourceState

	TerraformVersion      *version.Version
	TerraformVersionErr   error
	TerraformVersionState operation.OpState
}

func (m *StackRecord) Path() string {
	return m.path
}

func (m *StackRecord) Copy() *StackRecord {
	if m == nil {
		return nil
	}

	newRecord := &StackRecord{
		path:             m.path,
		Meta:             m.Meta.Copy(),
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

		for source, stacksDiags := range m.Diagnostics {
			newRecord.Diagnostics[source] = make(ast.Diagnostics, len(stacksDiags))

			for name, diags := range stacksDiags {
				newRecord.Diagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newRecord.Diagnostics[source][name], diags)
			}
		}
	}

	return newRecord
}

func newStack(modPath string) *StackRecord {
	return &StackRecord{
		path: modPath,
		DiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          operation.OpStateUnknown,
			globalAst.SchemaValidationSource:    operation.OpStateUnknown,
			globalAst.ReferenceValidationSource: operation.OpStateUnknown,
			globalAst.TerraformValidateSource:   operation.OpStateUnknown,
		},
	}
}
