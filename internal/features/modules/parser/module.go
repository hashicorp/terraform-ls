// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"log"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/modules/ast"
	"github.com/hashicorp/terraform-ls/internal/terraform/parser"
)

func ParseModuleFiles(fs parser.FS, modPath string) (ast.ModFiles, ast.ModDiags, error) {
	files := make(ast.ModFiles, 0)
	diags := make(ast.ModDiags, 0)

	log.Printf("Reading directory %s for module files", modPath)
	infos, err := fs.ReadDir(modPath)
	if err != nil {
		log.Printf("Error reading directory %s: %v", modPath, err)
		return nil, nil, err
	}

	log.Printf("Found %d entries in %s", len(infos), modPath)
	for _, info := range infos {
		if info.IsDir() {
			log.Printf("Skipping directory: %s", info.Name())
			continue
		}

		name := info.Name()
		log.Printf("Checking file: %s, is module file: %v", name, ast.IsModuleFilename(name))
		if !ast.IsModuleFilename(name) {
			continue
		}

		fullPath := filepath.Join(modPath, name)
		log.Printf("Reading module file: %s", fullPath)

		src, err := fs.ReadFile(fullPath)
		if err != nil {
			log.Printf("Error reading file %s: %v", fullPath, err)
			continue
		}

		filename := ast.ModFilename(name)
		f, pDiags := parser.ParseFile(src, filename)
		log.Printf("Parsed file %s: file nil? %v, diagnostics: %v", filename, f == nil, pDiags)

		diags[filename] = pDiags
		if f != nil {
			files[filename] = f
		}
	}

	log.Printf("Finished parsing module files. Found %d files", len(files))
	return files, diags, nil
}

func ParseModuleFile(fs parser.FS, filePath string) (*hcl.File, hcl.Diagnostics, error) {
	src, err := fs.ReadFile(filePath)
	if err != nil {
		// If a file isn't accessible, return
		return nil, nil, err
	}

	name := filepath.Base(filePath)
	filename := ast.ModFilename(name)

	f, pDiags := parser.ParseFile(src, filename)

	return f, pDiags, nil
}
