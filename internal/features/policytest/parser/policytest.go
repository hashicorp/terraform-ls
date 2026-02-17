// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/policytest/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/parser"
)

func ParsePolicyTestFiles(fs parser.FS, policytestPath string) (ast.PolicyTestFiles, ast.PolicyTestDiags, error) {
	files := make(ast.PolicyTestFiles, 0)
	diags := make(ast.PolicyTestDiags, 0)

	infos, err := fs.ReadDir(policytestPath)
	if err != nil {
		return nil, nil, err
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		if !ast.IsPolicyTestFilename(name) {
			continue
		}

		// TODO: overrides

		fullPath := filepath.Join(policytestPath, name)

		src, err := fs.ReadFile(fullPath)
		if err != nil {
			// If a file isn't accessible, continue with reading the
			// remaining policytest files
			continue
		}

		filename := ast.PolicyTestFilename(name)

		f, pDiags := parser.ParseFile(src, filename)

		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}

func ParsePolicyTestFile(fs parser.FS, filePath string) (*hcl.File, hcl.Diagnostics, error) {
	src, err := fs.ReadFile(filePath)
	if err != nil {
		// If a file isn't accessible, return
		return nil, nil, err
	}

	name := filepath.Base(filePath)
	filename := ast.PolicyTestFilename(name)

	f, pDiags := parser.ParseFile(src, filename)

	return f, pDiags, nil
}
