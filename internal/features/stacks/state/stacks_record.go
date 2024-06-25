// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	ParsedStackFiles      ast.StackFiles
	StackParsingErr       error
	StackDiagnostics      ast.SourceStackDiags
	StackDiagnosticsState globalAst.DiagnosticSourceState

	ParsedDeployFiles      ast.DeployFiles
	DeployParsingErr       error
	DeployDiagnostics      ast.SourceDeployDiags
	DeployDiagnosticsState globalAst.DiagnosticSourceState
}

func (m *StackRecord) Path() string {
	return m.path
}

func (m *StackRecord) Copy() *StackRecord {
	if m == nil {
		return nil
	}

	newRecord := &StackRecord{
		path:                   m.path,
		Meta:                   m.Meta.Copy(),
		StackParsingErr:        m.StackParsingErr,
		DeployParsingErr:       m.DeployParsingErr,
		StackDiagnosticsState:  m.StackDiagnosticsState.Copy(),
		DeployDiagnosticsState: m.DeployDiagnosticsState.Copy(),
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

	if m.ParsedDeployFiles != nil {
		newRecord.ParsedDeployFiles = make(ast.DeployFiles, len(m.ParsedDeployFiles))
		for name, f := range m.ParsedDeployFiles {
			// hcl.File is practically immutable once it comes out of parser
			newRecord.ParsedDeployFiles[name] = f
		}
	}

	if m.DeployDiagnostics != nil {
		newRecord.DeployDiagnostics = make(ast.SourceDeployDiags, len(m.DeployDiagnostics))

		for source, deployDiags := range m.DeployDiagnostics {
			newRecord.DeployDiagnostics[source] = make(ast.DeployDiags, len(deployDiags))

			for name, diags := range deployDiags {
				newRecord.DeployDiagnostics[source][name] = make(hcl.Diagnostics, len(diags))
				copy(newRecord.DeployDiagnostics[source][name], diags)
			}
		}
	}

	return newRecord
}

func newStack(stackPath string) *StackRecord {
	return &StackRecord{
		path: stackPath,
		StackDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          operation.OpStateUnknown,
			globalAst.SchemaValidationSource:    operation.OpStateUnknown,
			globalAst.ReferenceValidationSource: operation.OpStateUnknown,
			globalAst.TerraformValidateSource:   operation.OpStateUnknown,
		},
		DeployDiagnosticsState: globalAst.DiagnosticSourceState{
			globalAst.HCLParsingSource:          operation.OpStateUnknown,
			globalAst.SchemaValidationSource:    operation.OpStateUnknown,
			globalAst.ReferenceValidationSource: operation.OpStateUnknown,
			globalAst.TerraformValidateSource:   operation.OpStateUnknown,
		},
	}
}
