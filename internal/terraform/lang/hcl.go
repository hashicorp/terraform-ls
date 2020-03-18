package lang

import (
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
)

func jsonSchemaToHcl(js *tfjson.Schema) *hcl.BodySchema {
	hs := &hcl.BodySchema{
		Attributes: make([]hcl.AttributeSchema, 0),
		Blocks:     make([]hcl.BlockHeaderSchema, 0),
	}

	for name, attr := range js.Block.Attributes {
		hs.Attributes = append(hs.Attributes, hcl.AttributeSchema{
			Name:     name,
			Required: attr.Required,
		})
	}

	for name, _ := range js.Block.NestedBlocks {
		hs.Blocks = append(hs.Blocks, hcl.BlockHeaderSchema{
			Type: name,
		})

		// TODO: Deeply nested blocks
	}

	return hs
}

func bodyContainsPos(body *hclsyntax.Body, pos hcl.Pos) bool {
	rng := body.SrcRange

	// Account for the last character of header
	rng.Start.Byte += 1

	// Account for opening brace
	rng.Start.Byte += 1

	// Account for closing brace
	rng.End.Byte -= 1

	return rng.ContainsPos(pos)
}

func contentContainPos(body *hclsyntax.Body, pos hcl.Pos) bool {
	for _, attr := range body.Attributes {
		attrRange := attr.Range()

		// Account for the last character
		attrRange.End.Byte += 1

		if attrRange.ContainsPos(pos) {
			return true
		}
	}

	for _, block := range body.Blocks {
		blockRange := block.Range()

		// Account for the last character
		blockRange.End.Byte += 1

		if blockRange.ContainsPos(pos) {
			return true
		}
	}

	return false
}
