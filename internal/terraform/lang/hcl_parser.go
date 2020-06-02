package lang

import (
	"fmt"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
)

// ParseBlock parses HCL block's tokens based on tfjson's SchemaBlock
// and keeps hold of all tfjson schema details on block or attribute level
func ParseBlock(tBlock ihcl.TokenizedBlock, schema *tfjson.SchemaBlock) Block {
	// We ignore diags as we assume incomplete (invalid) configuration
	hclBlock, _ := hclsyntax.ParseBlockFromTokens(tBlock.Tokens())

	return parseBlock(hclBlock, schema)
}

func parseBlock(block *hclsyntax.Block, schema *tfjson.SchemaBlock) Block {
	b := &parsedBlock{
		hclBlock: block,
	}
	if block == nil {
		return b
	}

	body := block.Body

	if schema == nil {
		b.unknownAttributes = body.Attributes
		b.unknownBlocks = body.Blocks
		return b
	}

	b.AttributesMap, b.unknownAttributes = parseAttributes(body.Attributes, schema.Attributes)
	b.BlockTypesMap, b.unknownBlocks = parseBlockTypes(body.Blocks, schema.NestedBlocks)

	return b
}

// ParseLabels parses HCL block's tokens based on LabelSchema,
// returning labels as a slice of *ParsedLabel
func ParseLabels(tBlock ihcl.TokenizedBlock, schema LabelSchema) []*ParsedLabel {
	// We ignore diags as we assume incomplete (invalid) configuration
	hclBlock, _ := hclsyntax.ParseBlockFromTokens(tBlock.Tokens())

	return parseLabels(hclBlock, schema)
}

func parseLabels(block *hclsyntax.Block, schema LabelSchema) []*ParsedLabel {
	parsed := block.Labels

	labels := make([]*ParsedLabel, len(schema))

	for i, l := range schema {
		var value string
		var rng hcl.Range
		if len(parsed)-1 >= i {
			value = parsed[i]
			rng = block.LabelRanges[i]
		}
		labels[i] = &ParsedLabel{
			Name:  l.Name,
			Value: value,
			Range: rng,
		}
	}

	return labels
}

func LabelAtPos(labels []*ParsedLabel, pos hcl.Pos) (*ParsedLabel, bool) {
	for _, l := range labels {
		if rangeContainsOffset(l.Range, pos.Byte) {
			// TODO: Guard against crashes when user sets label where we don't expect it
			return l, true
		}
	}

	return nil, false
}

func AsHCLSyntaxBlock(block *hcl.Block) (*hclsyntax.Block, error) {
	if block == nil {
		return nil, nil
	}

	body, ok := block.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("invalid configuration format: %T", block.Body)
	}

	bodyRng := body.Range()

	openBraceRng := hcl.Range{
		Filename: bodyRng.Filename,
		Start:    bodyRng.Start,
		// hclsyntax.Body range always starts with open brace
		End: hcl.Pos{
			Column: bodyRng.Start.Column + 1,
			Byte:   bodyRng.Start.Byte + 1,
			Line:   bodyRng.Start.Line,
		},
	}
	closeBraceRng := hcl.Range{
		Filename: bodyRng.Filename,
		// hclsyntax.Body range always ends with close brace
		Start: hcl.Pos{
			Column: bodyRng.End.Column - 1,
			Byte:   bodyRng.End.Byte - 1,
			Line:   bodyRng.End.Line,
		},
		End: bodyRng.End,
	}

	return &hclsyntax.Block{
		Type:        block.Type,
		TypeRange:   block.TypeRange,
		Labels:      block.Labels,
		LabelRanges: block.LabelRanges,

		OpenBraceRange:  openBraceRng,
		CloseBraceRange: closeBraceRng,

		Body: body,
	}, nil
}
