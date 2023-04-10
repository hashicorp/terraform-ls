// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

func ParseModuleFiles(fs FS, modPath string) (ast.ModFiles, ast.ModDiags, error) {
	files := make(ast.ModFiles, 0)
	diags := make(ast.ModDiags, 0)

	infos, err := fs.ReadDir(modPath)
	if err != nil {
		return nil, nil, err
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		if !ast.IsModuleFilename(name) {
			continue
		}

		// TODO: overrides

		fullPath := filepath.Join(modPath, name)

		src, err := fs.ReadFile(fullPath)
		if err != nil {
			// If a file isn't accessible, continue with reading the
			// remaining module files
			continue
		}

		filename := ast.ModFilename(name)

		f, pDiags := parseFile(src, filename)

		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}
