package lang

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/sourcegraph/go-lsp"
	"github.com/zclconf/go-cty/cty"
)

func TestCompletableBlock_CompletionItemsAtPos(t *testing.T) {
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
	caps := &lsp.TextDocumentClientCapabilities{}
	caps.Completion.CompletionItem.SnippetSupport = true
	supportsSnippetsCapability := caps

	testCases := []struct {
		name string
		src  string
		pos  hcl.Pos
		sb   *tfjson.SchemaBlock
		caps *lsp.TextDocumentClientCapabilities

		expectedCandidates lsp.CompletionList
		expectedErr        error
	}{
		{
			"position in block body - no capabilities - attributes",
			`block "aws" {
  existing = "foo"
}`,
			hcl.Pos{Column: 1, Line: 2, Byte: 14},
			attrOnlySchema,
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
			"position in block body - no capabilities - single blocks",
			`block "aws" {
  declared_single {}
}`,
			hcl.Pos{Column: 1, Line: 2, Byte: 14},
			singleBlockOnlySchema,
			nil,
			lsp.CompletionList{
				IsIncomplete: false,
				Items: []lsp.CompletionItem{
					{
						Label:            "optional_single",
						Kind:             lsp.CIKField,
						Detail:           "(Optional, single)",
						InsertTextFormat: lsp.ITFPlainText,
					},
					{
						Label:            "required_single",
						Kind:             lsp.CIKField,
						Detail:           "(Required, single)",
						InsertTextFormat: lsp.ITFPlainText,
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
			nil,
			lsp.CompletionList{
				IsIncomplete: false,
				Items: []lsp.CompletionItem{
					{
						Label:            "optional_list",
						Kind:             lsp.CIKField,
						Detail:           "(Optional, list)",
						InsertTextFormat: lsp.ITFPlainText,
					},
					{
						Label:            "required_list",
						Kind:             lsp.CIKField,
						Detail:           "(Required, list)",
						InsertTextFormat: lsp.ITFPlainText,
					},
					{
						Label:            "undeclared_max1_list",
						Kind:             lsp.CIKField,
						Detail:           "(Optional, list)",
						InsertTextFormat: lsp.ITFPlainText,
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
			nil,
			lsp.CompletionList{
				Items: []lsp.CompletionItem{
					{
						Label:            "one",
						Kind:             lsp.CIKField,
						Detail:           "(Optional, string)",
						InsertTextFormat: lsp.ITFPlainText,
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
			block, err := AsHCLSyntaxBlock(parseHclBlock(t, tc.src))
			if err != nil {
				t.Fatal(err)
			}

			cb := &completableBlock{
				logger:   testLogger(),
				hclBlock: block,
			}

			if tc.caps != nil {
				cb.caps = *caps
			}

			if tc.sb != nil {
				cb.schema = tc.sb
			}

			list, err := cb.completionItemsAtPos(tc.pos)
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

			if diff := cmp.Diff(tc.expectedCandidates, list); diff != "" {
				t.Fatalf("Completion candidates don't match.\n%s", diff)
			}
		})
	}
}
