package lang

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/hashicorp/go-version"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/errors"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	lsp "github.com/sourcegraph/go-lsp"
)

// 0.12.0 first introduced HCL2 which provides
// more convenient/cleaner parsing
//
// We set no upper bound for now as there is only schema-related
// logic and schema format itself is version-checked elsewhere
//
// We may become more pessimistic as the parser begins to support
// language features which may differ between versions
// (e.g. meta-parameters)
const supportedVersion = ">= 0.12.0"

type ConfigBlock interface {
	CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error)
	Name() string
	BlockType() string
}

type configBlockFactory interface {
	New(*hcl.Block) (ConfigBlock, error)
	InitializeCapabilities(lsp.TextDocumentClientCapabilities)
}

type Parser interface {
	SetLogger(*log.Logger)
	SetCapabilities(lsp.TextDocumentClientCapabilities)
	SetSchemaReader(schema.Reader)
	ParseBlockFromHCL(*hcl.Block) (ConfigBlock, error)
}

type parser struct {
	logger *log.Logger
	caps   lsp.TextDocumentClientCapabilities

	schemaReader schema.Reader
}

func ParserSupportsTerraform(v string) error {
	tfVersion, err := version.NewVersion(v)
	if err != nil {
		return err
	}
	c, err := version.NewConstraint(supportedVersion)
	if err != nil {
		return err
	}

	if !c.Check(tfVersion) {
		return &errors.UnsupportedTerraformVersion{
			Component:   "parser",
			Version:     v,
			Constraints: c,
		}
	}

	return nil
}

// FindCompatibleParser finds a parser that is compatible with
// given Terraform version, so that it parses config accuretly
func FindCompatibleParser(v string) (Parser, error) {
	err := ParserSupportsTerraform(v)
	if err != nil {
		return nil, err
	}

	return newParser(), nil
}

func newParser() *parser {
	return &parser{
		logger: log.New(ioutil.Discard, "", 0),
	}
}

func (p *parser) SetLogger(logger *log.Logger) {
	p.logger = logger
}

func (p *parser) SetCapabilities(caps lsp.TextDocumentClientCapabilities) {
	p.caps = caps
}

func (p *parser) SetSchemaReader(sr schema.Reader) {
	p.schemaReader = sr
}

func (p *parser) blockTypes() map[string]configBlockFactory {
	return map[string]configBlockFactory{
		"provider": &providerBlockFactory{
			logger:       p.logger,
			schemaReader: p.schemaReader,
		},
		// "resource": ResourceBlock,
		// "data":     ResourceBlock,
		// "variable": VariableBlock,
		// "module":   ModuleBlock,
	}
}

func (p *parser) ParseBlockFromHCL(block *hcl.Block) (ConfigBlock, error) {
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
