package hcl

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/source"
	"github.com/pmezard/go-difflib/difflib"
)

func TestDiff(t *testing.T) {
	testCases := []struct {
		name                string
		beforeCfg, afterCfg string
		expectedChanges     filesystem.DocumentChanges
	}{
		{
			"no-op",
			`aaa
bbb
ccc`,
			`aaa
bbb
ccc`,
			filesystem.DocumentChanges{},
		},
		{
			"two separate lines replaced",
			`resource "aws_vpc" "name" {
  cidr_block = "sdf"
  tags = {
    "key" = "value"
    sdfasd = 1
    s = 3
  }
}`,
			`resource "aws_vpc" "name" {
  cidr_block = "sdf"
  tags = {
    "key"  = "value"
    sdfasd = 1
    s      = 3
  }
}`,
			filesystem.DocumentChanges{
				&fileChange{
					newText: `    "key"  = "value"
`,
					rng: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 4, Column: 1, Byte: 60},
						End:      hcl.Pos{Line: 5, Column: 1, Byte: 80},
					},
				},
				&fileChange{
					newText: `    s      = 3
`,
					rng: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 6, Column: 1, Byte: 95},
						End:      hcl.Pos{Line: 7, Column: 1, Byte: 105},
					},
				},
			},
		},
		{
			"whitespace shrinking",
			`resource "aws_vpc" "name" {
  cidr_block = "sdf"
  tags = {
    "key"  = "value"
    sdfasd = 1
    s      = 3


  }
}`,
			`resource "aws_vpc" "name" {
  cidr_block = "sdf"
  tags = {
    "key"  = "value"
    sdfasd = 1
    s      = 3
  }
}`,
			filesystem.DocumentChanges{
				&fileChange{
					newText: "",
					rng: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 7, Column: 1, Byte: 111},
						End:      hcl.Pos{Line: 9, Column: 1, Byte: 113},
					},
				},
			},
		},
		{
			"trailing whitespace removal",
			`resource "aws_vpc" "name" {
  
}`,
			`resource "aws_vpc" "name" {

}`,
			filesystem.DocumentChanges{
				&fileChange{
					newText: "\n",
					rng: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 2, Column: 1, Byte: 28},
						End:      hcl.Pos{Line: 3, Column: 1, Byte: 31},
					},
				},
			},
		},
		{
			"new line insertion",
			`resource "aws_vpc" "name" {}`,
			`resource "aws_vpc" "name" {
}`,
			filesystem.DocumentChanges{
				&fileChange{
					newText: `resource "aws_vpc" "name" {
}`,
					rng: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 29, Byte: 28},
					},
				},
			},
		},
		{
			"new line insertion at EOF",
			`resource "aws_vpc" "name" {
}
`,
			`resource "aws_vpc" "name" {
}

`,
			filesystem.DocumentChanges{
				&fileChange{
					newText: "\n",
					rng: hcl.Range{
						Start: hcl.Pos{Line: 3, Column: 1, Byte: 30},
						End:   hcl.Pos{Line: 3, Column: 1, Byte: 30},
					},
				},
			},
		},
		{
			"line insertion",
			`resource "aws_vpc" "name" {
  attr1 = "one"

  attr3 = "three"
}`,
			`resource "aws_vpc" "name" {
  attr1 = "one"
  attr2 = "two"
  attr3 = "three"
}`,
			filesystem.DocumentChanges{
				&fileChange{
					newText: `  attr2 = "two"
`,
					rng: hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 3, Column: 1, Byte: 44},
						End:      hcl.Pos{Line: 4, Column: 1, Byte: 45},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			linesBefore := source.MakeSourceLines("test.tf",
				[]byte(tc.beforeCfg))
			linesAfter := source.MakeSourceLines("test.tf",
				[]byte(tc.afterCfg))

			changes := diffLines("test.tf", linesBefore, linesAfter)

			opts := cmp.Options{
				cmp.AllowUnexported(fileChange{}),
				cmpopts.IgnoreTypes(difflib.OpCode{}),
			}

			if diff := cmp.Diff(tc.expectedChanges, changes, opts...); diff != "" {
				t.Fatalf("Changes don't match: %s", diff)
			}
		})
	}
}
