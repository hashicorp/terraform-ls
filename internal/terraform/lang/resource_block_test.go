package lang

import (
	"errors"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	"github.com/zclconf/go-cty/cty"
)

func TestResourceBlock_Name(t *testing.T) {
	testCases := []struct {
		name string
		src  string

		expectedName string
		expectedErr  error
	}{
		{
			"invalid config - no label",
			`resource {
}
`,
			"<unknown>",
			nil,
		},
		{
			"invalid config - single label",
			`resource "aws_instance" {
}
`,
			"aws_instance.<unknown>",
			nil,
		},
		{
			"invalid config - three labels",
			`resource "aws_instance" "name" "extra" {
}
`,
			"aws_instance.name",
			nil,
		},
		{
			"valid config",
			`resource "aws_instance" "name" {
}
`,
			"aws_instance.name",
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			block, err := AsHCLSyntaxBlock(parseHclBlock(t, tc.src))
			if err != nil {
				t.Fatal(err)
			}

			pf := &resourceBlockFactory{logger: log.New(os.Stdout, "", 0)}
			p, err := pf.New(block)

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

			name := p.Name()
			if name != tc.expectedName {
				t.Fatalf("Name doesn't match.\nexpected: %q\ngiven: %q",
					tc.expectedName, name)
			}
		})
	}
}

func TestResourceBlock_completionCandidatesAtPos(t *testing.T) {
	simpleSchema := &tfjson.ProviderSchemas{
		FormatVersion: "0.1",
		Schemas: map[string]*tfjson.ProviderSchema{
			"custom": {
				ResourceSchemas: map[string]*tfjson.Schema{
					"custom_rs": {
						Block: &tfjson.SchemaBlock{
							Attributes: map[string]*tfjson.SchemaAttribute{
								"attr_optional": {
									AttributeType: cty.String,
									Optional:      true,
								},
								"attr_required": {
									AttributeType: cty.String,
									Required:      true,
								},
							},
						},
					},
				},
			},
		},
	}
	testCases := []struct {
		name      string
		src       string
		schemas   *tfjson.ProviderSchemas
		readerErr error
		pos       hcl.Pos

		expectedCandidates []renderedCandidate
		expectedErr        error
	}{
		{
			"simple schema",
			`resource "custom_rs" "name" {

}`,
			simpleSchema,
			nil,
			hcl.Pos{Line: 2, Column: 1, Byte: 30},
			[]renderedCandidate{
				{
					Label:  "attr_optional",
					Detail: "(Optional, string)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 30},
						Text: `attr_optional = "${0:value}"`,
					},
				},
				{
					Label:  "attr_required",
					Detail: "(Required, string)",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 30},
						Text: `attr_required = "${0:value}"`,
					},
				},
			},
			nil,
		},
		{
			"missing type doesn't error",
			`resource "" "" {

}`,
			simpleSchema,
			nil,
			hcl.Pos{Line: 2, Column: 1, Byte: 17},
			[]renderedCandidate{},
			nil,
		},
		{
			"schema reader error",
			`resource "custom_rs" "name" {

}`,
			nil,
			errors.New("error getting schema"),
			hcl.Pos{Line: 2, Column: 1, Byte: 30},
			[]renderedCandidate{},
			errors.New("error getting schema"),
		},
		{
			"resource names",
			`resource "" "" {

}`,
			simpleSchema,
			nil,
			hcl.Pos{Line: 1, Column: 9, Byte: 10},
			[]renderedCandidate{
				{
					Label:  "custom_rs",
					Detail: "custom",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 1, Column: 9, Byte: 10},
						Text: "custom_rs",
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

			pf := &resourceBlockFactory{
				logger: log.New(os.Stdout, "", 0),
				schemaReader: &schema.MockReader{
					ProviderSchemas:   tc.schemas,
					ResourceSchemaErr: tc.readerErr,
				},
			}
			p, err := pf.New(block)
			if err != nil {
				t.Fatal(err)
			}

			candidates, err := p.CompletionCandidatesAtPos(tc.pos)
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

			rendered := renderCandidates(candidates, tc.pos)
			if diff := cmp.Diff(tc.expectedCandidates, rendered); diff != "" {
				t.Fatalf("Candidates don't match: %s", diff)
			}
		})
	}
}
