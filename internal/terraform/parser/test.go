// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

func ParseTestFiles(fs FS, modPath string) (ast.TestFiles, ast.TestDiags, error) {
	files := make(ast.TestFiles, 0)
	diags := make(ast.TestDiags, 0)

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
		if !ast.IsTestFilename(name) {
			continue
		}

		fullPath := filepath.Join(modPath, name)
		src, err := fs.ReadFile(fullPath)
		if err != nil {
			// If a file isn't accessible, continue with reading the
			// remaining module files
			continue
		}

		filename := ast.TestFilename(name)
		f, pDiags := parseFile(src, filename)

		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}
