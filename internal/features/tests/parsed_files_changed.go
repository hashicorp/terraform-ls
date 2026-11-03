// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"github.com/hashicorp/terraform-ls/internal/features/tests/ast"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfparser "github.com/hashicorp/terraform-ls/internal/terraform/parser"
)

// parsedFilesChanged reports whether the files parsed in path no longer match
// the files on disk or in open documents, so didOpen re-indexes only when
// something changed. It returns false until the first parse has finished, so
// didOpen does not schedule a second parse alongside it.
func (f *TestsFeature) parsedFilesChanged(path string) bool {
	record, err := f.store.TestRecordByPath(path)
	if err != nil {
		return false
	}
	if record.DiagnosticsState[globalAst.HCLParsingSource] != op.OpStateLoaded {
		return false
	}
	return !tfparser.FilesMatch(f.fs, path, isTestOrMockFilename, record.ParsedFiles)
}

func isTestOrMockFilename(name string) bool {
	return ast.IsTestFilename(name) || ast.IsMockFilename(name)
}
