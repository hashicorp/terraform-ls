package lang

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/hashicorp/go-version"
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/terraform/errors"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
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

type parser struct {
	logger *log.Logger

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

func (p *parser) SetSchemaReader(sr schema.Reader) {
	p.schemaReader = sr
}

func (p *parser) blockTypes() map[string]configBlockFactory {
	return map[string]configBlockFactory{
		"provider": &providerBlockFactory{
			logger:       p.logger,
			schemaReader: p.schemaReader,
		},
		"resource": &resourceBlockFactory{
			logger:       p.logger,
			schemaReader: p.schemaReader,
		},
		"data": &datasourceBlockFactory{
			logger:       p.logger,
			schemaReader: p.schemaReader,
		},
	}
}

func (p *parser) BlockTypeCandidates() CompletionCandidates {
	bTypes := p.blockTypes()

	list := &completeList{
		candidates: make([]CompletionCandidate, 0),
	}

	for name, t := range bTypes {
		list.candidates = append(list.candidates, &completableBlockType{
			TypeName:      name,
			LabelSchema:   t.LabelSchema(),
			documentation: t.Documentation(),
		})
	}

	return list
}

type completableBlockType struct {
	TypeName      string
	LabelSchema   LabelSchema
	documentation MarkupContent
}

func (bt *completableBlockType) Label() string {
	return bt.TypeName
}

func (bt *completableBlockType) Snippet(pos hcl.Pos) (hcl.Pos, string) {
	return pos, snippetForBlock(bt.TypeName, bt.LabelSchema)
}

func (bt *completableBlockType) Detail() string {
	return ""
}

func (bt *completableBlockType) Documentation() MarkupContent {
	return bt.documentation
}

func (p *parser) ParseBlockFromTokens(tokens hclsyntax.Tokens) (ConfigBlock, error) {
	if len(tokens) == 0 {
		return nil, EmptyConfigErr
	}

	// It is probably excessive to be parsing the whole block just for type
	// but there is no avoiding it without refactoring the upstream HCL parser
	// and it should not hurt the performance too much
	block, _ := hclsyntax.ParseBlockFromTokens(tokens)

	p.logger.Printf("Parsed block type: %q", block.Type)

	f, ok := p.blockTypes()[block.Type]
	if !ok {
		return nil, &unknownBlockTypeErr{block.Type}
	}

	cfgBlock, err := f.New(tokens)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", block.Type, err)
	}

	return cfgBlock, nil
}

func discardLog() *log.Logger {
	return log.New(ioutil.Discard, "", 0)
}
