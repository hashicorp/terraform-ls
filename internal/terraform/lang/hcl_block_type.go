package lang

import (
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
)

func (b *BlockType) Schema() *tfjson.SchemaBlockType {
	return b.schema
}

func (b *BlockType) BlockAtPos(pos hcl.Pos) (Block, bool) {
	for _, block := range b.BlockList {
		if b, ok := block.BlockAtPos(pos); ok {
			return b, true
		}
	}

	return nil, false
}

func (b *BlockType) PosInAttribute(pos hcl.Pos) bool {
	for _, block := range b.BlockList {
		if block.PosInAttribute(pos) {
			return true
		}
	}

	return false
}

func (b *BlockType) ReachedMaxItems() bool {
	blockS := b.schema

	declaredBlocks := len(b.BlockList)

	switch blockS.NestingMode {
	case tfjson.SchemaNestingModeSingle:
		if declaredBlocks > 0 {
			return true
		}
	case tfjson.SchemaNestingModeList, tfjson.SchemaNestingModeSet:
		if blockS.MaxItems > 0 && declaredBlocks >= int(blockS.MaxItems) {
			return true
		}
	}

	return false
}

type BlockTypes map[string]*BlockType

func (bt BlockTypes) AddBlock(name string, block *hclsyntax.Block, typeSchema *tfjson.SchemaBlockType) {
	_, ok := bt[name]
	if !ok {
		bt[name] = &BlockType{
			schema:    typeSchema,
			BlockList: make([]Block, 0),
		}
	}

	if block != nil {
		// SDK doesn't support named blocks yet, so we expect no labels here for now
		bt[name].BlockList = append(bt[name].BlockList, parseBlock(block, typeSchema.Block))
	}
}

func parseBlockTypes(blocks hclsyntax.Blocks, schemas map[string]*tfjson.SchemaBlockType) (BlockTypes, hclsyntax.Blocks) {
	var blockTypes BlockTypes = make(map[string]*BlockType, 0)
	remainingBlocks := blocks

	for name, typeSchema := range schemas {
		var matchingBlocks hclsyntax.Blocks
		matchingBlocks, remainingBlocks = blocksOfType(remainingBlocks, name)
		if len(matchingBlocks) > 0 {
			for _, b := range matchingBlocks {
				blockTypes.AddBlock(name, b, typeSchema)
			}
		} else {
			blockTypes.AddBlock(name, nil, typeSchema)
		}
	}

	return blockTypes, remainingBlocks
}

func blocksOfType(blocks hclsyntax.Blocks, bType string) (hclsyntax.Blocks, hclsyntax.Blocks) {
	matched := make([]*hclsyntax.Block, 0)
	remaining := make([]*hclsyntax.Block, 0)
	for _, b := range blocks {
		if b.Type == bType {
			matched = append(matched, b)
			continue
		}
		remaining = append(remaining, b)
	}

	return matched, remaining
}
