package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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
			return nil, nil, err
		}

		f, pDiags := hclsyntax.ParseConfig(src, name, hcl.InitialPos)
		filename := ast.ModFilename(name)
		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}
