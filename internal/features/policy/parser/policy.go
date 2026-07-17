// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/policy/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/parser"
)

func ParsePolicyFiles(fs parser.FS, policyPath string) (ast.PolicyFiles, ast.PolicyDiags, error) {
	files := make(ast.PolicyFiles, 0)
	diags := make(ast.PolicyDiags, 0)

	infos, err := fs.ReadDir(policyPath)
	if err != nil {
		return nil, nil, err
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		if !ast.IsPolicyFilename(name) {
			continue
		}

		// TODO: overrides

		fullPath := filepath.Join(policyPath, name)

		src, err := fs.ReadFile(fullPath)
		if err != nil {
			// If a file isn't accessible, continue with reading the
			// remaining policy files
			continue
		}

		filename := ast.PolicyFilename(name)

		f, pDiags := parser.ParseFile(src, filename)

		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}

func ParsePolicyFile(fs parser.FS, filePath string) (*hcl.File, hcl.Diagnostics, error) {
	src, err := fs.ReadFile(filePath)
	if err != nil {
		// If a file isn't accessible, return
		return nil, nil, err
	}

	name := filepath.Base(filePath)
	filename := ast.PolicyFilename(name)

	f, pDiags := parser.ParseFile(src, filename)

	return f, pDiags, nil
}
