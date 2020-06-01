package lang

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/json"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

func TestParseBlock_attributesAndBlockTypes(t *testing.T) {
	testCases := []struct {
		name   string
		cfg    string
		schema *tfjson.SchemaBlock

		expectedAttributes map[string]*Attribute
		expectedBlockTypes map[string]*BlockType
	}{
		{
			"empty cfg, nil schema",
			"",
			nil,
			nil,
			nil,
		},
		{
			"empty block, nil schema",
			`myblock {}`,
			nil,
			nil,
			nil,
		},
		{
			"block with labels, root attributes",
			`myblock "one" "two" {
    known1 = "hello"
    unknown1 = 99
}`,
			&tfjson.SchemaBlock{
				Attributes: map[string]*tfjson.SchemaAttribute{
					"known1": {
						AttributeType: cty.String,
						Optional:      true,
					},
				},
			},
			map[string]*Attribute{
				"known1": {},
			},
			map[string]*BlockType{},
		},
		{
			"block with labels, root single blocks",
			`myblock "one" "two" {
    single_required {}
    unknown_block {}
    single_optional {
        answer = 42
        unknown = "test"
    }
    another_unknown_block {
        answer_is = "nothing"
    }
}`,
			&tfjson.SchemaBlock{
				NestedBlocks: map[string]*tfjson.SchemaBlockType{
					"single_required": {
						NestingMode: tfjson.SchemaNestingModeSingle,
						MinItems:    1,
					},
					"single_optional": {
						NestingMode: tfjson.SchemaNestingModeSingle,
						Block: &tfjson.SchemaBlock{
							Attributes: map[string]*tfjson.SchemaAttribute{
								"answer": {
									AttributeType: cty.Number,
									Optional:      true,
								},
							},
						},
					},
				},
			},
			map[string]*Attribute{},
			map[string]*BlockType{
				"single_required": {
					BlockList: []Block{
						&parsedBlock{},
					},
				},
				"single_optional": {
					BlockList: []Block{&parsedBlock{
						AttributesMap: map[string]*Attribute{
							"answer": {},
						},
						BlockTypesMap: map[string]*BlockType{},
					}},
					schema: &tfjson.SchemaBlockType{
						NestingMode: tfjson.SchemaNestingModeSingle,
						Block: &tfjson.SchemaBlock{
							Attributes: map[string]*tfjson.SchemaAttribute{
								"answer": {
									Optional: true,
								},
							},
						},
					},
				},
			},
		},
		{
			"block with labels, double nested single blocks",
			`myblock "one" "two" {
    parent {
        first_gen {
        	attr1 = 32
        	second_gen {
        		attr = 10
        	}
        	unknown_attr = true
        }
        unknown_gen {}
    }
    unknown_block {}
}`,
			&tfjson.SchemaBlock{
				NestedBlocks: map[string]*tfjson.SchemaBlockType{
					"parent": {
						NestingMode: tfjson.SchemaNestingModeSingle,
						Block: &tfjson.SchemaBlock{
							NestedBlocks: map[string]*tfjson.SchemaBlockType{
								"first_gen": {
									NestingMode: tfjson.SchemaNestingModeList,
									Block: &tfjson.SchemaBlock{
										Attributes: map[string]*tfjson.SchemaAttribute{
											"attr1": {
												AttributeType: cty.Number,
												Optional:      true,
											},
										},
										NestedBlocks: map[string]*tfjson.SchemaBlockType{
											"second_gen": {
												NestingMode: tfjson.SchemaNestingModeList,
												Block: &tfjson.SchemaBlock{
													Attributes: map[string]*tfjson.SchemaAttribute{
														"attr": {
															AttributeType: cty.Number,
															Optional:      true,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			map[string]*Attribute{},
			map[string]*BlockType{
				"parent": {
					BlockList: []Block{
						&parsedBlock{
							AttributesMap: map[string]*Attribute{},
							BlockTypesMap: map[string]*BlockType{
								"first_gen": {
									BlockList: []Block{
										&parsedBlock{
											AttributesMap: map[string]*Attribute{
												"attr1": {},
											},
											BlockTypesMap: map[string]*BlockType{
												"second_gen": {
													BlockList: []Block{
														&parsedBlock{
															AttributesMap: map[string]*Attribute{
																"attr": {},
															},
															BlockTypesMap: map[string]*BlockType{},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			"block with labels, root list blocks",
			`myblock "one" "two" {
    list_block {
        is_unknown = 9
    }
    list_block {
        answer = 16
        another = 32
    }
}`,
			&tfjson.SchemaBlock{
				NestedBlocks: map[string]*tfjson.SchemaBlockType{
					"list_block": {
						NestingMode: tfjson.SchemaNestingModeList,
						Block: &tfjson.SchemaBlock{
							Attributes: map[string]*tfjson.SchemaAttribute{
								"answer": {
									AttributeType: cty.Number,
									Optional:      true,
								},
							},
						},
					},
				},
			},
			map[string]*Attribute{},
			map[string]*BlockType{
				"list_block": {
					BlockList: []Block{
						&parsedBlock{
							AttributesMap: map[string]*Attribute{
								"answer": {},
							},
							BlockTypesMap: map[string]*BlockType{},
						},
						&parsedBlock{
							AttributesMap: map[string]*Attribute{
								"answer": {},
							},
							BlockTypesMap: map[string]*BlockType{},
						},
					},
				},
			},
		},
		{
			"block with labels, root set blocks",
			`myblock "one" "two" {
    set_block {
        is_unknown = 9
    }
    set_block {
        answer = 16
        another = 32
    }
}`,
			&tfjson.SchemaBlock{
				NestedBlocks: map[string]*tfjson.SchemaBlockType{
					"set_block": {
						NestingMode: tfjson.SchemaNestingModeList,
						Block: &tfjson.SchemaBlock{
							Attributes: map[string]*tfjson.SchemaAttribute{
								"answer": {
									AttributeType: cty.Number,
									Optional:      true,
								},
							},
						},
					},
				},
			},
			map[string]*Attribute{},
			map[string]*BlockType{
				"set_block": {
					BlockList: []Block{
						&parsedBlock{
							AttributesMap: map[string]*Attribute{
								"answer": {},
							},
							BlockTypesMap: map[string]*BlockType{},
						},
						&parsedBlock{
							AttributesMap: map[string]*Attribute{
								"answer": {},
							},
							BlockTypesMap: map[string]*BlockType{},
						},
					},
				},
			},
		},
	}

	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(
			Attribute{},
			BlockType{},
			parsedBlock{},
		),
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			block, err := AsHCLSyntaxBlock(parseHclBlock(t, tc.cfg))
			if err != nil {
				t.Fatal(err)
			}

			b := ParseBlock(block, []*ParsedLabel{}, tc.schema)

			if diff := cmp.Diff(tc.expectedAttributes, b.Attributes(), opts...); diff != "" {
				t.Fatalf("Attributes don't match.\n%s", diff)
			}

			if diff := cmp.Diff(tc.expectedBlockTypes, b.BlockTypes(), opts...); diff != "" {
				t.Fatalf("BlockTypes don't match.\n%s", diff)
			}
		})
	}
}

func TestBlock_BlockAtPos(t *testing.T) {
	schema := &tfjson.SchemaBlock{
		Attributes: map[string]*tfjson.SchemaAttribute{
			"known_attr": {
				AttributeType: cty.String,
				Optional:      true,
			},
		},
		NestedBlocks: map[string]*tfjson.SchemaBlockType{
			"known_block": {
				NestingMode: tfjson.SchemaNestingModeSingle,
				Block: &tfjson.SchemaBlock{
					Attributes: map[string]*tfjson.SchemaAttribute{
						"one_attr": {
							AttributeType: cty.String,
							Optional:      true,
						},
					},
					NestedBlocks: map[string]*tfjson.SchemaBlockType{
						"nested_known_block": {
							NestingMode: tfjson.SchemaNestingModeList,
							Block: &tfjson.SchemaBlock{
								Attributes: map[string]*tfjson.SchemaAttribute{
									"nestedattr": {
										AttributeType: cty.String,
										Optional:      true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	testCases := []struct {
		name string
		cfg  string
		pos  hcl.Pos

		expectedBlock Block
	}{
		{
			"top-level block",
			`topblock "label1" {
  known_attr = "test"
  known_block {
    one_attr = "testvalue"
  }
  unknown_block{
    attr = "random"
  }
}`,
			hcl.Pos{Line: 2, Column: 1, Byte: 20},
			&parsedBlock{
				AttributesMap: map[string]*Attribute{
					"known_attr": {},
				},
				BlockTypesMap: map[string]*BlockType{
					"known_block": {
						BlockList: []Block{
							&parsedBlock{
								AttributesMap: map[string]*Attribute{
									"one_attr": {},
								},
								BlockTypesMap: map[string]*BlockType{
									"nested_known_block": {
										BlockList: []Block{},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			"second-level known block",
			`topblock "label1" {
  known_attr = "test"
  known_block {
    one_attr = "testvalue"
  }
  unknown_block{
    attr = "random"
  }
}`,
			hcl.Pos{Line: 4, Column: 3, Byte: 60},
			&parsedBlock{
				AttributesMap: map[string]*Attribute{
					"one_attr": {},
				},
				BlockTypesMap: map[string]*BlockType{
					"nested_known_block": {
						BlockList: []Block{},
					},
				},
			},
		},
		{
			"second-level unknown block",
			`topblock "label1" {
  known_attr = "test"
  known_block {
    one_attr = "testvalue"
  }
  unknown_block{
    attr = "random"
  }
}`,
			hcl.Pos{Line: 7, Column: 3, Byte: 108},
			nil,
		},
		{
			"third-level known block",
			`topblock "label1" {
  known_attr = "test"
  known_block {
    one_attr = "testvalue"
    nested_known_block {
      nestedattr = "test"
    }
  }
  unknown_block{
    attr = "random"
  }
}`,
			hcl.Pos{Line: 6, Column: 7, Byte: 116},
			&parsedBlock{
				AttributesMap: map[string]*Attribute{
					"nestedattr": {},
				},
				BlockTypesMap: map[string]*BlockType{},
			},
		},
	}

	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(
			Attribute{},
			BlockType{},
			parsedBlock{},
		),
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			block, err := AsHCLSyntaxBlock(parseHclBlock(t, tc.cfg))
			if err != nil {
				t.Fatal(err)
			}
			b := ParseBlock(block, []*ParsedLabel{}, schema)

			fBlock, _ := b.BlockAtPos(tc.pos)
			if diff := cmp.Diff(tc.expectedBlock, fBlock, opts...); diff != "" {
				t.Fatalf("Block doesn't match.\n%s", diff)
			}
		})
	}
}

func TestBlock_PosInBody(t *testing.T) {
	schema := &tfjson.SchemaBlock{
		Attributes: map[string]*tfjson.SchemaAttribute{
			"known_attr": {
				AttributeType: cty.String,
				Optional:      true,
			},
		},
		NestedBlocks: map[string]*tfjson.SchemaBlockType{
			"known_block": {
				NestingMode: tfjson.SchemaNestingModeSingle,
				Block: &tfjson.SchemaBlock{
					Attributes: map[string]*tfjson.SchemaAttribute{
						"one_attr": {
							AttributeType: cty.String,
							Optional:      true,
						},
					},
				},
			},
		},
	}
	testCases := []struct {
		name     string
		cfg      string
		pos      hcl.Pos
		expected bool
	}{
		{
			"in top-level type",
			`topblock "onelabel" {
}`,
			hcl.Pos{Column: 3, Line: 1, Byte: 4},
			false,
		},
		{
			"in top-level label",
			`topblock "onelabel" {
}`,
			hcl.Pos{Column: 13, Line: 1, Byte: 12},
			false,
		},
		{
			"in top-level open brace",
			`topblock "onelabel" {
}`,
			hcl.Pos{Column: 20, Line: 1, Byte: 19},
			false,
		},
		{
			"in top-level close brace",
			`topblock "onelabel" {
}`,
			hcl.Pos{Column: 1, Line: 2, Byte: 22},
			false,
		},
		{
			"in top-level body",
			`topblock "onelabel" {
}`,
			hcl.Pos{Column: 22, Line: 1, Byte: 21},
			true,
		},
		{
			"in known nested block type",
			`topblock "onelabel" {
  known_block {
    one_attr = "only"
  }
}`,
			hcl.Pos{Column: 5, Line: 2, Byte: 26},
			false,
		},
		{
			"in known nested block open brace",
			`topblock "onelabel" {
  known_block {
    one_attr = "only"
  }
}`,
			hcl.Pos{Column: 15, Line: 2, Byte: 36},
			false,
		},
		{
			"in known nested block close brace",
			`topblock "onelabel" {
  known_block {
    one_attr = "only"
  }
}`,
			hcl.Pos{Column: 1, Line: 4, Byte: 62},
			false,
		},
		{
			"in known nested block body",
			`topblock "onelabel" {
  known_block {
    one_attr = "only"
  }
}`,
			hcl.Pos{Column: 1, Line: 3, Byte: 38},
			true,
		},
		{
			"in unknown nested block type",
			`topblock "onelabel" {
  unknown_block {
    random = "something"
  }
}`,
			hcl.Pos{Column: 4, Line: 2, Byte: 26},
			false,
		},
		{
			"in unknown nested block open brace",
			`topblock "onelabel" {
  unknown_block {
    random = "something"
  }
}`,
			hcl.Pos{Column: 16, Line: 2, Byte: 38},
			false,
		},
		{
			"in unknown nested block close brace",
			`topblock "onelabel" {
  unknown_block {
    random = "something"
  }
}`,
			hcl.Pos{Column: 3, Line: 4, Byte: 67},
			false,
		},
		{
			"in unknown nested block body",
			`topblock "onelabel" {
  unknown_block {
    random = "something"
  }
}`,
			hcl.Pos{Column: 1, Line: 3, Byte: 40},
			true,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			block, err := AsHCLSyntaxBlock(parseHclBlock(t, tc.cfg))
			if err != nil {
				t.Fatal(err)
			}
			b := ParseBlock(block, []*ParsedLabel{}, schema)

			isInBody := b.PosInBody(tc.pos)
			if tc.expected != isInBody {
				if tc.expected {
					t.Fatalf("Expected position %#v to be in body", tc.pos)
				}
				t.Fatalf("Not expected position %#v to be in body", tc.pos)
			}
		})
	}
}

func TestBlock_PosInAttributes(t *testing.T) {
	schema := &tfjson.SchemaBlock{
		Attributes: map[string]*tfjson.SchemaAttribute{
			"known_attr": {
				AttributeType: cty.String,
				Optional:      true,
			},
		},
		NestedBlocks: map[string]*tfjson.SchemaBlockType{
			"known_block": {
				NestingMode: tfjson.SchemaNestingModeSingle,
				Block: &tfjson.SchemaBlock{
					Attributes: map[string]*tfjson.SchemaAttribute{
						"one_attr": {
							AttributeType: cty.String,
							Optional:      true,
						},
					},
				},
			},
		},
	}
	testCases := []struct {
		name     string
		cfg      string
		pos      hcl.Pos
		expected bool
	}{
		{
			"in known top-level attribute",
			`topblock "onelabel" {
  known_attr = "blah"
}`,
			hcl.Pos{Line: 2, Column: 6, Byte: 27},
			true,
		},
		{
			"end of known top-level attribute",
			`topblock "onelabel" {
  known_attr = "blah"
}`,
			hcl.Pos{Line: 2, Column: 22, Byte: 43},
			true,
		},
		{
			"outside top-level attribute",
			`topblock "onelabel" {
  known_attr = "blah"
}`,
			hcl.Pos{Line: 2, Column: 1, Byte: 22},
			false,
		},
		{
			"in known nested block attribute",
			`topblock "onelabel" {
  known_block {
    one_attr = "test"
  }
}`,
			hcl.Pos{Line: 3, Column: 8, Byte: 45},
			true,
		},
		{
			"end of known nested block attribute",
			`topblock "onelabel" {
  known_block {
    one_attr = "test"
  }
}`,
			hcl.Pos{Line: 3, Column: 22, Byte: 59},
			true,
		},
		{
			"outside known nested block attribute",
			`topblock "onelabel" {
  known_block {
    one_attr = "test"
  }
}`,
			hcl.Pos{Line: 3, Column: 3, Byte: 40},
			false,
		},
		{
			"in unknown nested block attribute",
			`topblock "onelabel" {
  unknown_block {
    attr = "test"
  }
}`,
			hcl.Pos{Line: 3, Column: 7, Byte: 46},
			true,
		},
		{
			"end of unknown nested block attribute",
			`topblock "onelabel" {
  unknown_block {
    attr = "test"
  }
}`,
			hcl.Pos{Line: 3, Column: 17, Byte: 56},
			// Column: 18, Byte: 57 (end of line) should work too
			// but we have no easy way to account for last chars
			// in unknown blocks
			true,
		},
		{
			"outside unknown nested block attribute",
			`topblock "onelabel" {
  unknown_block {
    attr = "test"
  }
}`,
			hcl.Pos{Line: 3, Column: 3, Byte: 42},
			false,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			block, err := AsHCLSyntaxBlock(parseHclBlock(t, tc.cfg))
			if err != nil {
				t.Fatal(err)
			}
			b := ParseBlock(block, []*ParsedLabel{}, schema)

			isInAttribute := b.PosInAttribute(tc.pos)
			if tc.expected != isInAttribute {
				if tc.expected {
					t.Fatalf("Expected position %#v to be in attribute", tc.pos)
				}
				t.Fatalf("Not expected position %#v to be in attribute", tc.pos)
			}
		})
	}
}

func TestAsHCLSyntaxBlock_invalid(t *testing.T) {
	jsonCfg := `{"blocktype": {}}`
	f, diags := json.Parse([]byte(jsonCfg), "/test.tf.json")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	schema := &hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{
		Type: "blocktype",
	}}}

	content, _, diags := f.Body.PartialContent(schema)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	_, err := AsHCLSyntaxBlock(content.Blocks[0])
	if err == nil {
		t.Fatal("Expected JSON configuration to be invalid")
	}
}

func TestAsHCLSyntaxBlock_valid(t *testing.T) {
	cfg := `provider "currywurst" {
  location = "Breda"
}`
	block := parseHclBlock(t, cfg)
	syntaxBlock, err := AsHCLSyntaxBlock(block)
	if err != nil {
		t.Fatal(err)
	}
	expectedBlock := &hclsyntax.Block{
		Type:   "provider",
		Labels: []string{"currywurst"},
		Body:   &hclsyntax.Body{},
		TypeRange: hcl.Range{
			Filename: "/test.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
			End:      hcl.Pos{Line: 1, Column: 9, Byte: 8},
		},
		LabelRanges: []hcl.Range{
			{
				Filename: "/test.tf",
				Start:    hcl.Pos{Line: 1, Column: 10, Byte: 9},
				End:      hcl.Pos{Line: 1, Column: 22, Byte: 21},
			},
		},
		OpenBraceRange: hcl.Range{
			Filename: "/test.tf",
			Start:    hcl.Pos{Line: 1, Column: 23, Byte: 22},
			End:      hcl.Pos{Line: 1, Column: 24, Byte: 23},
		},
		CloseBraceRange: hcl.Range{
			Filename: "/test.tf",
			Start:    hcl.Pos{Line: 3, Column: 1, Byte: 45},
			End:      hcl.Pos{Line: 3, Column: 2, Byte: 46},
		},
	}

	opts := []cmp.Option{
		cmpopts.IgnoreFields(hclsyntax.Block{}, "Body"),
	}

	if diff := cmp.Diff(expectedBlock, syntaxBlock, opts...); diff != "" {
		t.Fatalf("Block doesn't match.\n%s", diff)
	}
}
