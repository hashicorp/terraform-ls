package lang

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	"github.com/sourcegraph/go-lsp"
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
			"empty config",
			``,
			"",
			EmptyConfigErr(),
		},
		{
			"invalid config - two labels",
			`provider "aws" "extra" {
}
`,
			"",
			&InvalidLabelsErr{"provider", []string{"aws", "extra"}},
		},
		{
			"invalid config - no labels",
			`provider {
}
`,
			"",
			&InvalidLabelsErr{"provider", []string{}},
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
			block := parseHclBlock(t, tc.src)
			pf := &providerBlockFactory{logger: log.New(os.Stdout, "", 0)}
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

func TestProviderBlock_CompletionItemsAtPos(t *testing.T) {
	awsSchemas := &tfjson.ProviderSchema{
		ConfigSchema: &tfjson.Schema{
			Block: &tfjson.SchemaBlock{
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
			},
		},
	}
	caps := &lsp.TextDocumentClientCapabilities{}
	caps.Completion.CompletionItem.SnippetSupport = true
	supportsSnippetsCapability := caps

	testCases := []struct {
		name string
		src  string
		pos  hcl.Pos
		ps   *tfjson.ProviderSchemas
		caps *lsp.TextDocumentClientCapabilities

		expectedCandidates lsp.CompletionList
		expectedErr        error
	}{
		{
			"no schema",
			`provider "aws" {
}`,
			hcl.Pos{},
			nil,
			nil,
			lsp.CompletionList{},
			&schema.SchemaUnavailableErr{
				BlockType: "provider",
				FullName:  "aws",
			},
		},
		{
			"position in block type",
			`provider "aws" {
}`,
			hcl.Pos{Column: 3, Line: 1, Byte: 4},
			&tfjson.ProviderSchemas{
				Schemas: map[string]*tfjson.ProviderSchema{
					"aws": awsSchemas,
				},
			},
			nil,
			lsp.CompletionList{},
			nil,
		},
		{
			"position in block label",
			`provider "aws" {
}`,
			hcl.Pos{Column: 13, Line: 1, Byte: 12},
			&tfjson.ProviderSchemas{
				Schemas: map[string]*tfjson.ProviderSchema{
					"aws": awsSchemas,
				},
			},
			nil,
			lsp.CompletionList{},
			nil,
		},
		{
			"position in the middle of existing attribute",
			`provider "aws" {
  meh = "boo"
}`,
			hcl.Pos{Column: 4, Line: 2, Byte: 20},
			&tfjson.ProviderSchemas{
				Schemas: map[string]*tfjson.ProviderSchema{
					"aws": awsSchemas,
				},
			},
			nil,
			lsp.CompletionList{},
			nil,
		},
		{
			"position in block body - no capabilities",
			`provider "aws" {
  existing = "foo"
}`,
			hcl.Pos{Column: 1, Line: 2, Byte: 17},
			&tfjson.ProviderSchemas{
				Schemas: map[string]*tfjson.ProviderSchema{
					"aws": awsSchemas,
				},
			},
			nil,
			lsp.CompletionList{
				IsIncomplete: false,
				Items: []lsp.CompletionItem{
					{
						Label:            "first_str",
						Kind:             lsp.CIKField,
						Detail:           "(Optional, string)",
						InsertTextFormat: lsp.ITFPlainText,
					},
					{
						Label:            "required_bool",
						Kind:             lsp.CIKField,
						Detail:           "(Required, bool) test boolean",
						InsertTextFormat: lsp.ITFPlainText,
					},
					{
						Label:            "second_num",
						Kind:             lsp.CIKField,
						Detail:           "(Optional, number) random number",
						InsertTextFormat: lsp.ITFPlainText,
					},
				},
			},
			nil,
		},
		{
			"position in block body - snippet capable",
			`provider "aws" {
  existing = "foo"
}`,
			hcl.Pos{Column: 1, Line: 2, Byte: 17},
			&tfjson.ProviderSchemas{
				Schemas: map[string]*tfjson.ProviderSchema{
					"aws": awsSchemas,
				},
			},
			supportsSnippetsCapability,
			lsp.CompletionList{
				IsIncomplete: false,
				Items: []lsp.CompletionItem{
					{
						Label:            "first_str",
						Kind:             lsp.CIKField,
						Detail:           "(Optional, string)",
						InsertTextFormat: lsp.ITFSnippet,
						TextEdit: &lsp.TextEdit{
							Range: lsp.Range{
								Start: lsp.Position{Line: 1, Character: 0},
								End:   lsp.Position{Line: 1, Character: 0},
							},
							NewText: `first_str = "${0:value}"`,
						},
					},
					{
						Label:            "required_bool",
						Kind:             lsp.CIKField,
						Detail:           "(Required, bool) test boolean",
						InsertTextFormat: lsp.ITFSnippet,
						TextEdit: &lsp.TextEdit{
							Range: lsp.Range{
								Start: lsp.Position{Line: 1, Character: 0},
								End:   lsp.Position{Line: 1, Character: 0},
							},
							NewText: `required_bool = ${0:false}`,
						},
					},
					{
						Label:            "second_num",
						Kind:             lsp.CIKField,
						Detail:           "(Optional, number) random number",
						InsertTextFormat: lsp.ITFSnippet,
						TextEdit: &lsp.TextEdit{
							Range: lsp.Range{
								Start: lsp.Position{Line: 1, Character: 0},
								End:   lsp.Position{Line: 1, Character: 0},
							},
							NewText: `second_num = ${0:42}`,
						},
					},
				},
			},
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			block := parseHclBlock(t, tc.src)

			pf := &providerBlockFactory{}
			if tc.caps != nil {
				pf.InitializeCapabilities(*caps)
			}

			sr := schema.MockStorage(tc.ps)
			pf.schemaReader = sr

			p, err := pf.New(block)
			if err != nil {
				t.Fatal(err)
			}

			list, err := p.CompletionItemsAtPos(tc.pos)
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

			if diff := cmp.Diff(list, tc.expectedCandidates); diff != "" {
				t.Fatalf("Completion candidates don't match.\n%s", diff)
			}
		})
	}
}
