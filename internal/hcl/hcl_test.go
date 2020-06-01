package hcl

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
)

func TestFile_BlockAtPosition(t *testing.T) {
	testCases := []struct {
		name string

		content string
		pos     hcl.Pos

		expectedErr    error
		expectedTokens []hclsyntax.Token
	}{
		{
			"invalid config",
			`provider "aws" {`,
			hcl.Pos{
				Line:   1,
				Column: 1,
				Byte:   0,
			},
			nil, // Expect errors to be ignored
			[]hclsyntax.Token{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("provider"),
				},
				{
					Type:  hclsyntax.TokenOQuote,
					Bytes: []byte(`"`),
				},
				{
					Type:  hclsyntax.TokenQuotedLit,
					Bytes: []byte("aws"),
				},
				{
					Type:  hclsyntax.TokenCQuote,
					Bytes: []byte(`"`),
				},
				{
					Type:  hclsyntax.TokenOBrace,
					Bytes: []byte("{"),
				},
				{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				},
			},
		},
		{
			"valid config and position",
			`provider "aws" {

}
`,
			hcl.Pos{
				Line:   2,
				Column: 1,
				Byte:   17,
			},
			nil,
			[]hclsyntax.Token{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("provider"),
				},
				{
					Type:  hclsyntax.TokenOQuote,
					Bytes: []byte(`"`),
				},
				{
					Type:  hclsyntax.TokenQuotedLit,
					Bytes: []byte("aws"),
				},
				{
					Type:  hclsyntax.TokenCQuote,
					Bytes: []byte(`"`),
				},
				{
					Type:  hclsyntax.TokenOBrace,
					Bytes: []byte("{"),
				},
				{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				},
				{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				},
				{
					Type:  hclsyntax.TokenCBrace,
					Bytes: []byte("}"),
				},
				{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				},
			},
		},
		{
			"empty config and valid position",
			``,
			hcl.Pos{
				Line:   1,
				Column: 1,
				Byte:   0,
			},
			&NoBlockFoundErr{AtPos: hcl.Pos{Line: 1, Column: 1, Byte: 0}},
			nil,
		},
		{
			"empty config and out-of-range negative position",
			``,
			hcl.Pos{
				Line:   -42,
				Column: -3,
				Byte:   -46,
			},
			&InvalidHclPosErr{
				Pos:     hcl.Pos{Line: -42, Column: -3, Byte: -46},
				InRange: hcl.Range{Filename: "test.tf", Start: hcl.InitialPos, End: hcl.InitialPos},
			},
			nil,
		},
		{
			"empty config and out-of-range positive position",
			``,
			hcl.Pos{
				Line:   42,
				Column: 3,
				Byte:   46,
			},
			&InvalidHclPosErr{
				Pos:     hcl.Pos{Line: 42, Column: 3, Byte: 46},
				InRange: hcl.Range{Filename: "test.tf", Start: hcl.InitialPos, End: hcl.InitialPos},
			},
			nil,
		},
		{
			"valid config and out-of-range positive position",
			`provider "aws" {

}
`,
			hcl.Pos{
				Line:   42,
				Column: 3,
				Byte:   46,
			},
			&InvalidHclPosErr{
				Pos: hcl.Pos{Line: 42, Column: 3, Byte: 46},
				InRange: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.InitialPos,
					End:      hcl.Pos{Column: 1, Line: 4, Byte: 20},
				},
			},
			nil,
		},
		{
			"valid config and EOF position",
			`provider "aws" {

}
`,
			hcl.Pos{
				Line:   4,
				Column: 1,
				Byte:   20,
			},
			&NoBlockFoundErr{AtPos: hcl.Pos{Line: 4, Column: 1, Byte: 20}},
			nil,
		},
		{
			"valid config with newline near beginning",
			`
provider "aws" {
}`,
			hcl.Pos{
				Line:   2,
				Column: 1,
				Byte:   1,
			},
			nil, // Expect errors to be ignored
			[]hclsyntax.Token{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("provider"),
				},
				{
					Type:  hclsyntax.TokenOQuote,
					Bytes: []byte(`"`),
				},
				{
					Type:  hclsyntax.TokenQuotedLit,
					Bytes: []byte("aws"),
				},
				{
					Type:  hclsyntax.TokenCQuote,
					Bytes: []byte(`"`),
				},
				{
					Type:  hclsyntax.TokenOBrace,
					Bytes: []byte("{"),
				},
				{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				},
				{
					Type:  hclsyntax.TokenCBrace,
					Bytes: []byte("}"),
				},
				{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				},
			},
		},
	}

	opts := cmp.Options{
		cmpopts.IgnoreFields(hclsyntax.Token{}, "Range"),
		cmpopts.IgnoreFields(hcl.Diagnostic{}, "Subject"),
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i+1, tc.name), func(t *testing.T) {
			fsFile := filesystem.NewFile("test.tf", []byte(tc.content))
			f := NewFile(fsFile)

			tokens, err := f.BlockTokensAtPosition(tc.pos)
			if err != nil {
				if tc.expectedErr == nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(tc.expectedErr, err, opts...); diff != "" {
					t.Fatalf("Error mismatch: %s", diff)
				}
				return
			}
			if tc.expectedErr != nil {
				t.Fatalf("Expected error: %s", tc.expectedErr)
			}

			if diff := cmp.Diff(hclsyntax.Tokens(tc.expectedTokens), tokens, opts...); diff != "" {
				t.Fatalf("Unexpected token difference: %s", diff)
			}

		})
	}
}

type testPosition struct {
	filesystem.FileHandler
	pos hcl.Pos
}

func (p *testPosition) Position() hcl.Pos {
	return p.pos
}
