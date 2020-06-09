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
					Label:         "Fill Required Fields...",
					Detail:        "",
					Documentation: "Auto-generated object literal (required fields)\n{\n\trequired_bool = false\n}",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: `required_bool = ${1:false}`,
					},
				},
				{
					Label:         "first_str",
					Detail:        "Optional, string",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: `first_str = "${1:value}"`,
					},
				},
				{
					Label:         "required_bool",
					Detail:        "Required, bool",
					Documentation: "test boolean",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "required_bool = ${1:false}",
					},
				},
				{
					Label:         "second_num",
					Detail:        "Optional, number",
					Documentation: "random number",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "second_num = ${1:0}",
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
					Label:         "Fill Required Fields...",
					Detail:        "",
					Documentation: "Auto-generated object literal (required fields)\n{\n\t\n\trequired_single {\n\t  \n\t}\n}",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "required_single {\n  ${1}\n}\n",
					},
				},
				{
					Label:         "optional_single",
					Detail:        "Block, single",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "optional_single {\n  ${1}\n}",
					},
				},
				{
					Label:         "required_single",
					Detail:        "Block, single, min: 1",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "required_single {\n  ${1}\n}",
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
					Label:         "Fill Required Fields...",
					Detail:        "",
					Documentation: "Auto-generated object literal (required fields)\n{\n\t\n\trequired_list {\n\t  \n\t}\n}",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "required_list {\n  ${1}\n}\n",
					},
				},
				{
					Label:         "optional_list",
					Detail:        "Block, list",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "optional_list {\n  ${1}\n}",
					},
				},
				{
					Label:         "required_list",
					Detail:        "Block, list, min: 1",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "required_list {\n  ${1}\n}",
					},
				},
				{
					Label:         "undeclared_max1_list",
					Detail:        "Block, list, max: 1",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "undeclared_max1_list {\n  ${1}\n}",
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
					Label:         "one",
					Detail:        "Optional, string",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 20, Byte: 33},
						Text: `one = "${1:value}"`,
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
					Label:         "Fill Required Fields...",
					Detail:        "",
					Documentation: "Auto-generated object literal (required fields)\n{\n\trequired_bool = false\n}",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 14},
						Text: "required_bool = ${1:false}",
					},
				},
				{
					Label:         "first_str",
					Detail:        "Optional, string",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Column: 1, Line: 2, Byte: 14},
						Text: `first_str = "${1:value}"`,
					},
				},
				{
					Label:         "required_bool",
					Detail:        "Required, bool",
					Documentation: "test boolean",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Column: 1, Line: 2, Byte: 14},
						Text: `required_bool = ${1:false}`,
					},
				},
				{
					Label:         "second_num",
					Detail:        "Optional, number",
					Documentation: "random number",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Column: 1, Line: 2, Byte: 14},
						Text: `second_num = ${1:0}`,
					},
				},
			},
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			tBlock := newTestBlock(t, tc.src)

			cb := &completableBlock{
				logger:       testLogger(),
				parsedLabels: []*ParsedLabel{},
				schema:       tc.sb,
				tBlock:       tBlock,
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

func TestCompletableLabels_CompletionCandidatesAtPos_overLimit(t *testing.T) {
	tBlock := newTestBlock(t, `provider "" {
}`)

	cl := &completableLabels{
		logger: testLogger(),
		parsedLabels: []*ParsedLabel{
			{Name: "type", Range: hcl.Range{
				Filename: "/test.tf",
				Start:    hcl.Pos{Line: 1, Column: 10, Byte: 9},
				End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
			}},
		},
		tBlock: tBlock,
		labels: map[string][]*labelCandidate{
			"type": []*labelCandidate{
				{label: "aaa"},
				{label: "bbb"},
				{label: "ccc"},
			},
		},
		maxCandidates: 1,
	}
	c, err := cl.completionCandidatesAtPos(hcl.Pos{Line: 1, Column: 11, Byte: 10})
	if err != nil {
		t.Fatal(err)
	}

	if c.Len() != 1 {
		t.Fatalf("Expected exactly 1 candidate, %d given", c.Len())
	}

	if c.IsComplete() {
		t.Fatalf("Expected list of 3 with maxCandidates=1 to be incomplete")
	}
}

func TestCompletableLabels_CompletionCandidatesAtPos_matchingLimit(t *testing.T) {
	tBlock := newTestBlock(t, `provider "" {
}`)

	cl := &completableLabels{
		logger: testLogger(),
		parsedLabels: []*ParsedLabel{
			{Name: "type", Range: hcl.Range{
				Filename: "/test.tf",
				Start:    hcl.Pos{Line: 1, Column: 10, Byte: 9},
				End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
			}},
		},
		tBlock: tBlock,
		labels: map[string][]*labelCandidate{
			"type": []*labelCandidate{
				{label: "aaa"},
				{label: "bbb"},
			},
		},
		maxCandidates: 2,
	}
	c, err := cl.completionCandidatesAtPos(hcl.Pos{Line: 1, Column: 11, Byte: 10})
	if err != nil {
		t.Fatal(err)
	}

	if c.Len() != 2 {
		t.Fatalf("Expected exactly 2 candidates, %d given", c.Len())
	}

	if !c.IsComplete() {
		t.Fatalf("Expected list of 2 with maxCandidates=2 to be complete")
	}
}

func TestCompletableLabels_CompletionCandidatesAtPos_withPrefix(t *testing.T) {
	tBlock := newTestBlock(t, `resource "prov_xyz" {
}`)

	cl := &completableLabels{
		logger: testLogger(),
		parsedLabels: []*ParsedLabel{
			{Name: "type", Range: hcl.Range{
				Filename: "/test.tf",
				Start:    hcl.Pos{Line: 1, Column: 10, Byte: 9},
				End:      hcl.Pos{Line: 1, Column: 20, Byte: 19},
			}},
		},
		tBlock: tBlock,
		labels: map[string][]*labelCandidate{
			"type": []*labelCandidate{
				{label: "prov_aaa"},
				{label: "prov_bbb"},
				{label: "ccc"},
			},
		},
	}
	c, err := cl.completionCandidatesAtPos(hcl.Pos{Line: 1, Column: 16, Byte: 15})
	if err != nil {
		t.Fatal(err)
	}

	if c.Len() != 2 {
		t.Fatalf("Expected exactly 2 candidate, %d given", c.Len())
	}

	candidates := c.List()
	te := candidates[0].PlainText()
	expectedTextEdit := &textEdit{
		newText: "prov_aaa",
		rng: &hcl.Range{
			Filename: "/test.tf",
			Start: hcl.Pos{
				Line:   1,
				Column: 11,
				Byte:   10,
			},
			End: hcl.Pos{
				Line:   1,
				Column: 19,
				Byte:   18,
			},
		},
	}

	opts := cmp.AllowUnexported(textEdit{})
	if diff := cmp.Diff(expectedTextEdit, te, opts); diff != "" {
		t.Fatalf("Text edit doesn't match: %s", diff)
	}
}

func renderCandidates(list CompletionCandidates, pos hcl.Pos) []renderedCandidate {
	if list == nil {
		return []renderedCandidate{}
	}
	rendered := make([]renderedCandidate, len(list.List()))
	for i, c := range list.List() {
		text := c.Snippet()
		doc := ""
		if c.Documentation() != nil {
			doc = c.Documentation().Value()
		}

		rendered[i] = renderedCandidate{
			Label:         c.Label(),
			Detail:        c.Detail(),
			Documentation: doc,
			Snippet: renderedSnippet{
				Pos:  pos,
				Text: text.NewText(),
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
	Label         string
	Detail        string
	Documentation string
	Snippet       renderedSnippet
}

type renderedSnippet struct {
	Pos  hcl.Pos
	Text string
}
