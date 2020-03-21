package lang

import (
	"fmt"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

func TestParseBlockTypes_basic(t *testing.T) {
	cfg := `
custom_block "one" {
}
custom_block "two" {
  region = "us-west"
}
meh "boo" {
}
custom_block "three" {
}
`
	typeSchemas := map[string]*tfjson.SchemaBlockType{
		"custom_block": {
			NestingMode: tfjson.SchemaNestingModeList,
			Block: &tfjson.SchemaBlock{
				Attributes: map[string]*tfjson.SchemaAttribute{
					"region": {
						AttributeType: cty.String,
						Optional:      true,
					},
				},
			},
		},
	}
	blocks := parseHclSyntaxBlocks(t, cfg)

	blockTypes, unknownBlocks := parseBlockTypes(blocks, typeSchemas)

	bt, ok := blockTypes["custom_block"]
	if !ok {
		t.Fatalf("Block type %q not found", "custom_block")
	}

	expectedBlockCount := 3
	if len(bt.BlockList) != expectedBlockCount {
		t.Fatalf("Block count mismatch.\nExpected: %d\nGiven: %d\n",
			expectedBlockCount, len(bt.BlockList))
	}

	if len(unknownBlocks) != 1 {
		t.Fatalf("Unknown block mismatch.\nExpected: 1\nGiven: %d\n",
			len(unknownBlocks))
	}
}

func TestParseBlockTypes_undeclared(t *testing.T) {
	cfg := `
custom_block "one" {
}
`
	typeSchemas := map[string]*tfjson.SchemaBlockType{
		"undeclared_block": {
			NestingMode: tfjson.SchemaNestingModeList,
			Block: &tfjson.SchemaBlock{
				Attributes: map[string]*tfjson.SchemaAttribute{
					"region": {
						AttributeType: cty.String,
						Optional:      true,
					},
				},
			},
		},
	}
	blocks := parseHclSyntaxBlocks(t, cfg)

	blockTypes, unknownBlocks := parseBlockTypes(blocks, typeSchemas)

	bt, ok := blockTypes["undeclared_block"]
	if !ok {
		t.Fatalf("Block type %q not found", "undeclared_block")
	}

	expectedBlockCount := 0
	if len(bt.BlockList) != expectedBlockCount {
		t.Fatalf("Block count mismatch.\nExpected: %d\nGiven: %d\n",
			expectedBlockCount, len(bt.BlockList))
	}

	if len(unknownBlocks) != 1 {
		t.Fatalf("Unknown block mismatch.\nExpected: 1\nGiven: %d\n",
			len(unknownBlocks))
	}
}

func TestBlockTypeReachedMaxItems(t *testing.T) {
	testCases := []struct {
		schema *tfjson.SchemaBlockType
		blocks []Block

		expectReachedItems bool
	}{
		{
			&tfjson.SchemaBlockType{
				NestingMode: tfjson.SchemaNestingModeSingle,
			},
			[]Block{},
			false,
		},
		{
			&tfjson.SchemaBlockType{
				NestingMode: tfjson.SchemaNestingModeSingle,
			},
			[]Block{&parsedBlock{}},
			true,
		},
		{
			&tfjson.SchemaBlockType{
				NestingMode: tfjson.SchemaNestingModeList,
			},
			[]Block{&parsedBlock{}, &parsedBlock{}},
			false,
		},
		{
			&tfjson.SchemaBlockType{
				NestingMode: tfjson.SchemaNestingModeList,
				MaxItems:    2,
			},
			[]Block{&parsedBlock{}, &parsedBlock{}},
			true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			bType := &BlockType{
				schema:    tc.schema,
				BlockList: tc.blocks,
			}
			reached := bType.ReachedMaxItems()
			if reached != tc.expectReachedItems {
				if tc.expectReachedItems {
					t.Fatalf("Expected max items to be reached for %#v",
						tc.schema)
				}
				t.Fatalf("Expected max items NOT to be reached for %#v",
					tc.schema)
			}
		})
	}
}
