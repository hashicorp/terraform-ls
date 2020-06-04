package lang

import (
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type parsedBlock struct {
	hclBlock      *hclsyntax.Block
	AttributesMap map[string]*Attribute
	BlockTypesMap map[string]*BlockType

	unknownAttributes hclsyntax.Attributes
	unknownBlocks     hclsyntax.Blocks
}

func (b *parsedBlock) Attributes() map[string]*Attribute {
	return b.AttributesMap
}

func (b *parsedBlock) BlockTypes() map[string]*BlockType {
	return b.BlockTypesMap
}

func (b *parsedBlock) BlockAtPos(pos hcl.Pos) (string, Block, bool) {
	// Check nested blocks first
	for _, nbt := range b.BlockTypesMap {
		if ty, b, ok := nbt.BlockAtPos(pos); ok {
			return ty, b, true
		}
	}

	// Check unknown blocks to prevent false positive below
	for _, ub := range b.unknownBlocks {
		if ub.Range().ContainsPos(pos) {
			return ub.Type, nil, false
		}
	}

	if b.hclBlock.Range().ContainsPos(pos) {
		return b.hclBlock.Type, b, true
	}

	return "", nil, false
}

func (b *parsedBlock) Range() hcl.Range {
	return b.hclBlock.Range()
}

func (b *parsedBlock) PosInBody(pos hcl.Pos) bool {
	for _, blockType := range b.BlockTypesMap {
		for _, b := range blockType.BlockList {
			if b.Range().ContainsPos(pos) {
				return true
			}
		}
	}

	for _, ub := range b.unknownBlocks {
		if ub.Range().ContainsPos(pos) {
			return true
		}
	}

	if b.hclBlock == nil || b.hclBlock.Body == nil {
		return false
	}

	return posInBodyOfBlock(b.hclBlock, pos)
}

func posInBodyOfBlock(block *hclsyntax.Block, pos hcl.Pos) bool {
	return block.Body.Range().ContainsPos(pos) &&
		!block.OpenBraceRange.ContainsPos(pos) &&
		!block.CloseBraceRange.ContainsPos(pos)
}

func (b *parsedBlock) PosInAttribute(pos hcl.Pos) bool {
	for _, attr := range b.AttributesMap {
		if !attr.IsDeclared() {
			continue
		}

		// Account for the last character
		if rangeContainsOffset(attr.Range(), pos.Byte) {
			return true
		}
	}

	for _, nbt := range b.BlockTypesMap {
		if nbt.PosInAttribute(pos) {
			return true
		}
	}

	// Checking unknown attributes is somewhat pointless
	// in the context of LS (as we can't autocomplete these)
	// but we do it anyway, for "correctness"

	for _, attr := range b.unknownAttributes {
		// Account for the last character
		if rangeContainsOffset(attr.Range(), pos.Byte) {
			return true
		}
	}

	for _, block := range b.unknownBlocks {
		if block.Body.AttributeAtPos(pos) != nil {
			return true
		}
	}

	return false
}
