// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/variables/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/parser"
)

func ParseVariableFiles(fs parser.FS, modPath string) (ast.VarsFiles, ast.VarsDiags, error) {
	files := make(ast.VarsFiles, 0)
	diags := make(ast.VarsDiags, 0)

	dirEntries, err := fs.ReadDir(modPath)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			// We only care about files
			continue
		}

		name := entry.Name()
		if !ast.IsVarsFilename(name) {
			continue
		}

		fullPath := filepath.Join(modPath, name)

		src, err := fs.ReadFile(fullPath)
		if err != nil {
			return nil, nil, err
		}

		filename := ast.VarsFilename(name)

		f, pDiags := parser.ParseFile(src, filename)

		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}

func ParseVariableFile(fs parser.FS, filePath string) (*hcl.File, hcl.Diagnostics, error) {
	src, err := fs.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}

	name := filepath.Base(filePath)
	filename := ast.VarsFilename(name)

	f, pDiags := parser.ParseFile(src, filename)

	return f, pDiags, nil
}
