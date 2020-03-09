package filesystem

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/sourcegraph/go-lsp"
)

func TestHclBlockAtDocPosition(t *testing.T) {
	testCases := []struct {
		name string

		content string
		pos     lsp.Position

		expectedErr   error
		expectedBlock *hcl.Block
		expectedPos   hcl.Pos
	}{
		{
			"valid config and position",
			`provider "aws" {

}
`,
			lsp.Position{
				Line:      1,
				Character: 0,
			},
			nil,
			&hcl.Block{
				Type:   "provider",
				Labels: []string{"aws"},
			},
			hcl.Pos{Line: 2, Column: 1, Byte: 17},
		},
		{
			"empty config and valid position",
			``,
			lsp.Position{
				Line:      0,
				Character: 0,
			},
			&NoBlockFoundErr{AtPos: hcl.Pos{Line: 1, Column: 1, Byte: 0}},
			nil,
			hcl.Pos{Line: 1, Column: 1, Byte: 0},
		},
		{
			"empty config and out-of-range negative position",
			``,
			lsp.Position{
				Line:      -42,
				Character: -3,
			},
			&InvalidLspPosErr{Pos: lsp.Position{Line: -42, Character: -3}},
			nil,
			hcl.Pos{},
		},
		{
			"empty config and out-of-range positive position",
			``,
			lsp.Position{
				Line:      42,
				Character: 3,
			},
			&InvalidLspPosErr{Pos: lsp.Position{Line: 42, Character: 3}},
			nil,
			hcl.Pos{},
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
			&InvalidLspPosErr{Pos: lsp.Position{Line: 42, Character: 3}},
			nil,
			hcl.Pos{},
		},
	}

	opts := cmpopts.IgnoreFields(hcl.Block{},
		"Body", "DefRange", "TypeRange", "LabelRanges")

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i+1, tc.name), func(t *testing.T) {
			fs := NewFilesystem()
			fs.SetLogger(log.New(os.Stdout, "", 0))

			uri := lsp.DocumentURI("file:///test.tf")
			fs.Open(lsp.TextDocumentItem{
				URI:        uri,
				LanguageID: "terraform",
				Text:       tc.content,
				Version:    0,
			})

			block, pos, err := fs.HclBlockAtDocPosition(lsp.TextDocumentPositionParams{
				TextDocument: lsp.TextDocumentIdentifier{
					URI: uri,
				},
				Position: tc.pos,
			})
			if err != nil {
				if tc.expectedErr == nil {
					t.Fatal(err)
				}
				if err.Error() != tc.expectedErr.Error() {
					t.Fatalf("Unexpected error.\nexpected: %#v\ngiven:    %#v\n",
						tc.expectedErr, err)
				}
			}

			if diff := cmp.Diff(block, tc.expectedBlock, opts); diff != "" {
				t.Fatalf("Unexpected block difference: %s", diff)
			}
			if diff := cmp.Diff(pos, tc.expectedPos, opts); diff != "" {
				t.Fatalf("Unexpected pos difference: %s", diff)
			}

		})
	}
}
