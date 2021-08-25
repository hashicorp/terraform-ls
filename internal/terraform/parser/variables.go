package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

func ParseVariableFiles(fs FS, modPath string) (ast.VarsFiles, ast.VarsDiags, error) {
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

		f, pDiags := hclsyntax.ParseConfig(src, name, hcl.InitialPos)
		filename := ast.VarsFilename(name)
		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}
