package lang

import (
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type parsedBlock struct {
	hclBlock      *hclsyntax.Block
	labels        []*Label
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

func (b *parsedBlock) BlockAtPos(pos hcl.Pos) (Block, bool) {
	// Check nested blocks first
	for _, nbt := range b.BlockTypesMap {
		if b, ok := nbt.BlockAtPos(pos); ok {
			return b, true
		}
	}

	// Check unknown blocks to prevent false positive below
	for _, ub := range b.unknownBlocks {
		if ub.Range().ContainsPos(pos) {
			return nil, false
		}
	}

	if b.hclBlock.Range().ContainsPos(pos) {
		return b, true
	}

	return nil, false
}

func (b *parsedBlock) Range() hcl.Range {
	return b.hclBlock.Range()
}

func (b *parsedBlock) PosInBody(pos hcl.Pos) bool {
	for _, blockType := range b.BlockTypesMap {
		for _, b := range blockType.BlockList {
			if b.Range().ContainsPos(pos) {
				return b.PosInBody(pos)
			}
		}
	}

	for _, ub := range b.unknownBlocks {
		if ub.Range().ContainsPos(pos) {
			return posInBodyOfBlock(ub, pos)
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

		attrRange := attr.Range()
		// Account for the last character
		attrRange.End.Byte += 1

		if attrRange.ContainsPos(pos) {
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
		attrRange := attr.Range()

		// Account for the last character
		attrRange.End.Byte += 1

		if attrRange.ContainsPos(pos) {
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
