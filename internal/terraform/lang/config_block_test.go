package lang

import (
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

func TestCompletableBlock_CompletionCandidatesAtPos(t *testing.T) {
	attrOnlySchema := &tfjson.SchemaBlock{
		Attributes: map[string]*tfjson.SchemaAttribute{
			"first_str": {
				AttributeType: cty.String,
				Optional:      true,
			},
			"second_num": {
				AttributeType: cty.Number,
				Optional:      true,
				Description:   "random number",
			},
			"required_bool": {
				AttributeType: cty.Bool,
				Required:      true,
				Description:   "test boolean",
			},
			"computed_only": {
				AttributeType: cty.String,
				Computed:      true,
			},
			"existing": {
				AttributeType: cty.String,
				Optional:      true,
			},
		},
	}
	singleBlockOnlySchema := &tfjson.SchemaBlock{
		NestedBlocks: map[string]*tfjson.SchemaBlockType{
			"optional_single": {
				NestingMode: tfjson.SchemaNestingModeSingle,
			},
			"required_single": {
				NestingMode: tfjson.SchemaNestingModeSingle,
				MinItems:    1,
			},
			"declared_single": {
				NestingMode: tfjson.SchemaNestingModeSingle,
				Block: &tfjson.SchemaBlock{
					Attributes: map[string]*tfjson.SchemaAttribute{
						"one": {
							AttributeType: cty.String,
							Optional:      true,
						},
					},
				},
			},
		},
	}
	listBlockOnlySchema := &tfjson.SchemaBlock{
		NestedBlocks: map[string]*tfjson.SchemaBlockType{
			"optional_list": {
				NestingMode: tfjson.SchemaNestingModeList,
			},
			"required_list": {
				NestingMode: tfjson.SchemaNestingModeList,
				MinItems:    1,
			},
			"declared_max1_list": {
				NestingMode: tfjson.SchemaNestingModeList,
				MaxItems:    1,
			},
			"undeclared_max1_list": {
				NestingMode: tfjson.SchemaNestingModeList,
				MaxItems:    1,
			},
		},
	}

	testCases := []struct {
		name string
		src  string
		pos  hcl.Pos
		sb   *tfjson.SchemaBlock

		expectedCandidates []renderedCandidate
		expectedErr        error
	}{
		{
			"position in block body - no capabilities - attributes",
			`block "aws" {
  existing = "foo"
}`,
			hcl.Pos{Column: 1, Line: 2, Byte: 14},
			attrOnlySchema,
			[]renderedCandidate{
				{
					Label:  "first_str",
					Detail: "(Optional, string)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: `first_str = "${0:value}"`,
					},
				},
				{
					Label:  "required_bool",
					Detail: "(Required, bool) test boolean",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "required_bool = ${0:false}",
					},
				},
				{
					Label:  "second_num",
					Detail: "(Optional, number) random number",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "second_num = ${0:42}",
					},
				},
			},
			nil,
		},
		{
			"position in block body - no capabilities - single blocks",
			`block "aws" {
  declared_single {}
}`,
			hcl.Pos{Column: 1, Line: 2, Byte: 14},
			singleBlockOnlySchema,
			[]renderedCandidate{
				{
					Label:  "optional_single",
					Detail: "(Optional, single)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "optional_single {\n  ${0}\n}",
					},
				},
				{
					Label:  "required_single",
					Detail: "(Required, single)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "required_single {\n  ${0}\n}",
					},
				},
			},
			nil,
		},
		{
			"position in root block body - no capabilities - list blocks",
			`block "aws" {
  declared_max1_list {}
}`,
			hcl.Pos{Column: 1, Line: 2, Byte: 14},
			listBlockOnlySchema,
			[]renderedCandidate{
				{
					Label:  "optional_list",
					Detail: "(Optional, list)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "optional_list {\n  ${0}\n}",
					},
				},
				{
					Label:  "required_list",
					Detail: "(Required, list)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "required_list {\n  ${0}\n}",
					},
				},
				{
					Label:  "undeclared_max1_list",
					Detail: "(Optional, list)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "undeclared_max1_list {\n  ${0}\n}",
					},
				},
			},
			nil,
		},
		{
			"position in nested block's body",
			`block "aws" {
  declared_single { }
}`,
			hcl.Pos{Column: 20, Line: 2, Byte: 33},
			singleBlockOnlySchema,
			[]renderedCandidate{
				{
					Label:  "one",
					Detail: "(Optional, string)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 20, Byte: 33},
						Text: `one = "${0:value}"`,
					},
				},
			},
			nil,
		},
		{
			"position in root block body - snippet capable",
			`block "aws" {
  existing = "foo"
}`,
			hcl.Pos{Column: 1, Line: 2, Byte: 14},
			attrOnlySchema,
			[]renderedCandidate{
				{
					Label:  "first_str",
					Detail: "(Optional, string)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Column: 1, Line: 2, Byte: 14},
						Text: `first_str = "${0:value}"`,
					},
				},
				{
					Label:  "required_bool",
					Detail: "(Required, bool) test boolean",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Column: 1, Line: 2, Byte: 14},
						Text: `required_bool = ${0:false}`,
					},
				},
				{
					Label:  "second_num",
					Detail: "(Optional, number) random number",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Column: 1, Line: 2, Byte: 14},
						Text: `second_num = ${0:42}`,
					},
				},
			},
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			block, err := AsHCLSyntaxBlock(parseHclBlock(t, tc.src))
			if err != nil {
				t.Fatal(err)
			}

			cb := &completableBlock{
				logger: testLogger(),
				block:  ParseBlock(block, []*ParsedLabel{}, tc.sb),
			}

			list, err := cb.completionCandidatesAtPos(tc.pos)
			if err != nil {
				if tc.expectedErr != nil && err.Error() == tc.expectedErr.Error() {
					return
				}
				t.Fatalf("Errors don't match.\nexpected: %#v\ngiven: %#v",
					tc.expectedErr, err)
			}
			if tc.expectedErr != nil {
				t.Fatalf("Expected error: %#v", tc.expectedErr)
			}

			rendered := renderCandidates(list, tc.pos)

			if diff := cmp.Diff(tc.expectedCandidates, rendered); diff != "" {
				t.Fatalf("Completion candidates don't match.\n%s", diff)
			}
		})
	}
}

func renderCandidates(list CompletionCandidates, pos hcl.Pos) []renderedCandidate {
	if list == nil {
		return []renderedCandidate{}
	}
	rendered := make([]renderedCandidate, len(list.List()))
	for i, c := range list.List() {
		pos, text := c.Snippet(pos)

		rendered[i] = renderedCandidate{
			Label:  c.Label(),
			Detail: c.Detail(),
			Snippet: renderedSnippet{
				Pos:  pos,
				Text: text,
			},
		}
	}
	return rendered
}

func sortRenderedCandidates(candidates []renderedCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Label < candidates[j].Label
	})
}

type renderedCandidate struct {
	Label   string
	Detail  string
	Snippet renderedSnippet
}

type renderedSnippet struct {
	Pos  hcl.Pos
	Text string
}
