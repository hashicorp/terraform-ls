// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package policy

import (
	"github.com/hashicorp/terraform-ls/internal/features/policy/ast"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfparser "github.com/hashicorp/terraform-ls/internal/terraform/parser"
)

// parsedFilesChanged reports whether the files parsed in path no longer match
// the files on disk or in open documents, so didOpen re-indexes only when
// something changed. It returns false until the first parse has finished, so
// didOpen does not schedule a second parse alongside it.
func (f *PolicyFeature) parsedFilesChanged(path string) bool {
	record, err := f.Store.PolicyRecordByPath(path)
	if err != nil {
		return false
	}
	if record.PolicyDiagnosticsState[globalAst.HCLParsingSource] != op.OpStateLoaded {
		return false
	}
	return !tfparser.FilesMatch(f.fs, path, ast.IsPolicyFilename, record.ParsedPolicyFiles)
}
