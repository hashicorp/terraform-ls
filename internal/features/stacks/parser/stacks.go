package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/parser"
)

func ParseStackFiles(fs parser.FS, stacksPath string) (ast.StackFiles, ast.StackDiags, error) {
	files := make(ast.StackFiles, 0)
	diags := make(ast.StackDiags, 0)

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
		if !ast.IsStacksFilename(name) {
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

		filename := ast.StackFilename(name)

		f, pDiags := parser.ParseFile(src, filename)

		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	return files, diags, nil
}

func ParseStackFile(fs parser.FS, filePath string) (*hcl.File, hcl.Diagnostics, error) {
	src, err := fs.ReadFile(filePath)
	if err != nil {
		// If a file isn't accessible, return
		return nil, nil, err
	}

	name := filepath.Base(filePath)
	filename := ast.StackFilename(name)

	f, pDiags := parser.ParseFile(src, filename)

	return f, pDiags, nil
}
