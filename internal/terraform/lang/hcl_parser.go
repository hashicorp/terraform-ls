package lang

import (
	"fmt"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
)

// ParseBlock parses HCL configuration based on tfjson's SchemaBlock
// and keeps hold of all tfjson schema details on block or attribute level
func ParseBlock(tokens hclsyntax.Tokens, labels []*ParsedLabel, schema *tfjson.SchemaBlock) (Block, error) {
	hclBlock, _ := hclsyntax.ParseBlockFromTokens(tokens)

	return parseBlock(hclBlock, labels, schema), nil
}

func parseBlock(block *hclsyntax.Block, labels []*ParsedLabel, schema *tfjson.SchemaBlock) Block {
	b := &parsedBlock{
		hclBlock: block,
		labels:   labels,
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

func ParseLabels(tokens hclsyntax.Tokens, schema LabelSchema) ([]*ParsedLabel, error) {
	hclBlock, _ := hclsyntax.ParseBlockFromTokens(tokens)

	return parseLabels(hclBlock.Type, schema, hclBlock.Labels), nil
}

func parseLabels(blockType string, schema LabelSchema, parsed []string) []*ParsedLabel {
	labels := make([]*ParsedLabel, len(schema))

	for i, l := range schema {
		var value string
		if len(parsed)-1 >= i {
			value = parsed[i]
		}
		labels[i] = &ParsedLabel{
			Name:  l.Name,
			Value: value,
		}
	}

	return labels
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
