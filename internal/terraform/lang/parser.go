package lang

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/hashicorp/go-version"
	hcl "github.com/hashicorp/hcl/v2"
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
const parserVersionConstraint = ">= 0.12.0"

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
	c, err := version.NewConstraint(parserVersionConstraint)
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
			caps:         p.caps,
		},
		"resource": &resourceBlockFactory{
			logger:       p.logger,
			schemaReader: p.schemaReader,
			caps:         p.caps,
		},
		// "data":     ResourceBlock,
	}
}

func (p *parser) ParseBlockFromHCL(block *hcl.Block) (ConfigBlock, error) {
	if block == nil {
		return nil, EmptyConfigErr
	}

	f, ok := p.blockTypes()[block.Type]
	if !ok {
		return nil, &unknownBlockTypeErr{block.Type}
	}

	hsBlock, err := AsHCLSyntaxBlock(block)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", block.Type, err)
	}

	cfgBlock, err := f.New(hsBlock)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", hsBlock.Type, err)
	}

	return cfgBlock, nil
}

func discardLog() *log.Logger {
	return log.New(ioutil.Discard, "", 0)
}
