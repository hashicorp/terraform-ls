// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/tests/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/parser"
)

func ParseFiles(fs parser.FS, testPath string) (ast.Files, ast.Diagnostics, error) {
	files := make(ast.Files, 0)
	diags := make(ast.Diagnostics, 0)

	infos, err := fs.ReadDir(testPath)
	if err != nil {
		return nil, nil, err
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		if !ast.IsTestFilename(name) && !ast.IsMockFilename(name) {
			continue
		}

		fullPath := filepath.Join(testPath, name)

		src, err := fs.ReadFile(fullPath)
		if err != nil {
			// If a file isn't accessible, continue with reading the
			// remaining module files
			continue
		}

		filename := ast.FilenameFromName(name)
		f, pDiags := parser.ParseFile(src, filename)

		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}

func ParseFile(fs parser.FS, filePath string) (*hcl.File, hcl.Diagnostics, error) {
	src, err := fs.ReadFile(filePath)
	if err != nil {
		// If a file isn't accessible, return
		return nil, nil, err
	}

	name := filepath.Base(filePath)
	filename := ast.FilenameFromName(name)
	f, pDiags := parser.ParseFile(src, filename)

	return f, pDiags, nil
}
