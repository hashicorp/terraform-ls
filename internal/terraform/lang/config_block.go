package lang

import (
	"fmt"
	"io/ioutil"
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	lsp "github.com/sourcegraph/go-lsp"
)

type ConfigBlock interface {
	CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error)
	LoadSchema(ps *tfjson.ProviderSchemas) error
	Name() string
	BlockType() string
}

type configBlockFactory interface {
	New(*hcl.Block) (ConfigBlock, error)
	InitializeCapabilities(lsp.TextDocumentClientCapabilities)
}

type parser struct {
	logger *log.Logger
	caps   lsp.TextDocumentClientCapabilities
}

func NewParser(logger *log.Logger, caps lsp.TextDocumentClientCapabilities) *parser {
	return &parser{logger: logger, caps: caps}
}

func (p *parser) blockTypes() map[string]configBlockFactory {
	return map[string]configBlockFactory{
		"provider": &providerBlockFactory{logger: p.logger},
		// "resource": ResourceBlock,
		// "data":     ResourceBlock,
		// "variable": VariableBlock,
		// "module":   ModuleBlock,
	}
}

func (p *parser) ParseBlockFromHcl(block *hcl.Block) (ConfigBlock, error) {
	f, ok := p.blockTypes()[block.Type]
	if !ok {
		return nil, fmt.Errorf("unknown block type: %q", block.Type)
	}
	f.InitializeCapabilities(p.caps)

	cfgBlock, err := f.New(block)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", block.Type, err)
	}

	return cfgBlock, nil
}

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

func undeclaredSchemaAttributes(attrs map[string]*tfjson.SchemaAttribute,
	declared hcl.Attributes) map[string]*tfjson.SchemaAttribute {

	for name, _ := range attrs {
		if _, ok := declared[name]; ok {
			delete(attrs, name)
		}
	}

	return attrs
}

func emptyLogger() *log.Logger {
	return log.New(ioutil.Discard, "", 0)
}
