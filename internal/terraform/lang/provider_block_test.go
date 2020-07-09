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

func TestProviderBlock_Name(t *testing.T) {
	testCases := []struct {
		name string
		src  string

		expectedName string
		expectedErr  error
	}{
		{
			"invalid config - two labels",
			`provider "aws" "extra" {
}
`,
			"aws",
			nil,
		},
		{
			"invalid config - no labels",
			`provider {
}
`,
			"<unknown>",
			nil,
		},
		{
			"valid config",
			`provider "aws" {

}
`,
			"aws",
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			tBlock := newTestBlock(t, tc.src)

			pf := &providerBlockFactory{logger: log.New(os.Stdout, "", 0)}
			p, err := pf.New(tBlock)

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

func TestProviderBlock_completionCandidatesAtPos(t *testing.T) {
	simpleSchema := &tfjson.ProviderSchemas{
		FormatVersion: "0.1",
		Schemas: map[string]*tfjson.ProviderSchema{
			"custom": {
				ConfigSchema: &tfjson.Schema{
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
			`provider "custom" {

}`,
			simpleSchema,
			nil,
			hcl.Pos{Line: 2, Column: 1, Byte: 20},
			[]renderedCandidate{
				{
					Label:         "attr_optional",
					Detail:        "Optional, string",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 20},
						Text: `attr_optional = "${0:value}"`,
					},
				},
				{
					Label:         "attr_required",
					Detail:        "Required, string",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 2, Column: 1, Byte: 20},
						Text: `attr_required = "${0:value}"`,
					},
				},
			},
			nil,
		},
		{
			"wrong provider name",
			`provider "x" {
}`,
			simpleSchema,
			nil,
			hcl.Pos{Line: 2, Column: 1, Byte: 14},
			[]renderedCandidate{},
			&schema.SchemaUnavailableErr{BlockType: "provider", FullName: "x"},
		},
		{
			"schema reader error",
			`provider "custom" {

}`,
			nil,
			errors.New("error getting schema"),
			hcl.Pos{Line: 2, Column: 1, Byte: 20},
			[]renderedCandidate{},
			errors.New("error getting schema"),
		},
		{
			"provider names",
			`provider "" {

}`,
			simpleSchema,
			nil,
			hcl.Pos{Line: 1, Column: 9, Byte: 10},
			[]renderedCandidate{
				{
					Label:         "custom",
					Detail:        "hashicorp/custom",
					Documentation: "",
					Snippet: renderedSnippet{
						Pos:  hcl.Pos{Line: 1, Column: 9, Byte: 10},
						Text: "custom",
					},
				},
			},
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			tBlock := newTestBlock(t, tc.src)

			pf := &providerBlockFactory{
				logger: log.New(os.Stdout, "", 0),
				schemaReader: &schema.MockReader{
					ProviderSchemas:   tc.schemas,
					ProviderSchemaErr: tc.readerErr,
				},
			}
			p, err := pf.New(tBlock)
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
