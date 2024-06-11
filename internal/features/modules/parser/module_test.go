// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/modules/ast"
)

func TestParseModuleFiles(t *testing.T) {
	testCases := []struct {
		dirName           string
		expectedFileNames map[string]struct{}
		expectedDiags     map[string]hcl.Diagnostics
	}{
		{
			"empty-dir",
			map[string]struct{}{},
			map[string]hcl.Diagnostics{},
		},
		{
			"valid-mod-files",
			map[string]struct{}{
				"empty.tf":     {},
				"resources.tf": {},
			},
			map[string]hcl.Diagnostics{
				"empty.tf":     nil,
				"resources.tf": nil,
			},
		},
		{
			"valid-mod-files-with-extra-items",
			map[string]struct{}{
				".hidden.tf": {},
				"main.tf":    {},
			},
			map[string]hcl.Diagnostics{
				".hidden.tf": nil,
				"main.tf":    nil,
			},
		},
		{
			"invalid-mod-files",
			map[string]struct{}{
				"incomplete-block.tf": {},
				"missing-brace.tf":    {},
			},
			map[string]hcl.Diagnostics{
				"incomplete-block.tf": {
					{
						Severity: hcl.DiagError,
						Summary:  "Invalid block definition",
						Detail:   `A block definition must have block content delimited by "{" and "}", starting on the same line as the block header.`,
						Subject: &hcl.Range{
							Filename: "incomplete-block.tf",
							Start:    hcl.Pos{Line: 1, Column: 30, Byte: 29},
							End:      hcl.Pos{Line: 2, Column: 1, Byte: 30},
						},
						Context: &hcl.Range{
							Filename: "incomplete-block.tf",
							Start:    hcl.InitialPos,
							End:      hcl.Pos{Line: 2, Column: 1, Byte: 30},
						},
					},
				},
				"missing-brace.tf": {
					{
						Severity: hcl.DiagError,
						Summary:  "Unclosed configuration block",
						Detail:   "There is no closing brace for this block before the end of the file. This may be caused by incorrect brace nesting elsewhere in this file.",
						Subject: &hcl.Range{
							Filename: "missing-brace.tf",
							Start:    hcl.Pos{Line: 1, Column: 40, Byte: 39},
							End:      hcl.Pos{Line: 1, Column: 41, Byte: 40},
						},
					},
				},
			},
		},
		{
			"invalid-links",
			map[string]struct{}{
				"resources.tf": {},
			},
			map[string]hcl.Diagnostics{
				"resources.tf": nil,
			},
		},
	}

	fs := osFs{}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.dirName), func(t *testing.T) {
			modPath := filepath.Join("testdata", tc.dirName)

			files, diags, err := ParseModuleFiles(fs, modPath)
			if err != nil {
				t.Fatal(err)
			}

			fileNames := mapKeys(files)
			if diff := cmp.Diff(tc.expectedFileNames, fileNames); diff != "" {
				t.Fatalf("unexpected file names: %s", diff)
			}

			if diff := cmp.Diff(tc.expectedDiags, diags.AsMap()); diff != "" {
				t.Fatalf("unexpected diagnostics: %s", diff)
			}
		})
	}
}

func mapKeys(mf ast.ModFiles) map[string]struct{} {
	m := make(map[string]struct{}, len(mf))
	for name := range mf {
		m[name.String()] = struct{}{}
	}
	return m
}

type osFs struct{}

func (osfs osFs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (osfs osFs) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (osfs osFs) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (osfs osFs) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
