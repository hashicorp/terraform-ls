package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/parser"
)

func ParseDeployFiles(fs parser.FS, stacksPath string) (ast.DeployFiles, ast.DeployDiags, error) {
	files := make(ast.DeployFiles, 0)
	diags := make(ast.DeployDiags, 0)

	infos, err := fs.ReadDir(stacksPath)
	if err != nil {
		return nil, nil, err
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		if !ast.IsDeployFilename(name) {
			continue
		}

		// TODO: overrides

		fullPath := filepath.Join(stacksPath, name)

		src, err := fs.ReadFile(fullPath)
		if err != nil {
			// If a file isn't accessible, continue with reading the
			// remaining module files
			continue
		}

		filename := ast.DeployFilename(name)

		f, pDiags := parser.ParseFile(src, filename)

		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}

func ParseDeployFile(fs parser.FS, filePath string) (*hcl.File, hcl.Diagnostics, error) {
	src, err := fs.ReadFile(filePath)
	if err != nil {
		// If a file isn't accessible, return
		return nil, nil, err
	}

	name := filepath.Base(filePath)
	filename := ast.DeployFilename(name)

	f, pDiags := parser.ParseFile(src, filename)

	return f, pDiags, nil
}
