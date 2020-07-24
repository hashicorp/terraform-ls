package lsp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	lsp "github.com/sourcegraph/go-lsp"
)

func TestFile_LspPosToHCLPos(t *testing.T) {
	testCases := []struct {
		name string

		content        string
		lspPos         lsp.Position
		expectedHclPos hcl.Pos
		expectedErr    error
	}{
		{
			"empty config, valid position",
			``,
			lsp.Position{Character: 0, Line: 0},
			hcl.Pos{Column: 1, Line: 1, Byte: 0},
			nil,
		},
		{
			"valid config, valid position",
			`provider "aws" {

}
`,
			lsp.Position{Character: 0, Line: 1},
			hcl.Pos{Column: 1, Line: 2, Byte: 17},
			nil,
		},
		{
			"valid non-ASCII config, position before unicode char",
			`provider "aws" {
	special_region = "🙃"
}
`,
			lsp.Position{Character: 0, Line: 1},
			hcl.Pos{Column: 1, Line: 2, Byte: 17},
			nil,
		},
		{
			"valid non-ASCII config, position after unicode char",
			`provider "aws" {
	special_region = "🙃"
}
`,
			lsp.Position{Character: 22, Line: 1},
			hcl.Pos{Column: 23, Line: 2, Byte: 41},
			nil,
		},
		{
			"empty config and out-of-range negative position",
			``,
			lsp.Position{
				Line:      -42,
				Character: -3,
			},
			hcl.Pos{},
			&InvalidLspPosErr{Pos: lsp.Position{Line: -42, Character: -3}},
		},
		{
			"empty config and out-of-range positive position",
			``,
			lsp.Position{
				Line:      42,
				Character: 3,
			},
			hcl.Pos{},
			&InvalidLspPosErr{Pos: lsp.Position{Line: 42, Character: 3}},
		},
		{
			"valid config and out-of-range positive position",
			`provider "aws" {

}
`,
			lsp.Position{
				Line:      42,
				Character: 3,
			},
			hcl.Pos{},
			&InvalidLspPosErr{Pos: lsp.Position{Line: 42, Character: 3}},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			f := filesystem.NewDocumentMetadata(FileHandlerFromPath("/test.tf"), []byte(tc.content))

			hclPos, err := lspPositionToHCL(f.Lines(), tc.lspPos)
			if err != nil {
				if tc.expectedErr == nil {
					t.Fatal(err)
				}
				if err.Error() != tc.expectedErr.Error() {
					t.Fatalf("Unexpected error.\nexpected: %#v\ngiven:    %#v\n",
						tc.expectedErr, err)
				}
			}

			if hclPos != tc.expectedHclPos {
				t.Fatalf("HCL position didn't match.\nexpected: %#v\ngiven: %#v",
					tc.expectedHclPos, hclPos)
			}
		})
	}
}
