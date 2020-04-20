package lang

import (
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type providerBlockFactory struct {
	logger *log.Logger

	schemaReader schema.Reader
}

func (f *providerBlockFactory) New(block *hclsyntax.Block) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	return &providerBlock{
		logger: f.logger,

		labelSchema: LabelSchema{"name"},
		hclBlock:    block,
		sr:          f.schemaReader,
	}, nil
}

func (f *providerBlockFactory) BlockType() string {
	return "provider"
}

type providerBlock struct {
	logger *log.Logger

	labelSchema LabelSchema
	labels      []*Label
	hclBlock    *hclsyntax.Block
	sr          schema.Reader
}

func (p *providerBlock) Name() string {
	firstLabel := p.RawName()
	if firstLabel == "" {
		return "<unknown>"
	}
	return firstLabel
}

func (p *providerBlock) RawName() string {
	return p.Labels()[0].Value
}

func (p *providerBlock) Labels() []*Label {
	if p.labels != nil {
		return p.labels
	}
	p.labels = parseLabels(p.BlockType(), p.labelSchema, p.hclBlock.Labels)
	return p.labels
}

func (p *providerBlock) BlockType() string {
	return "provider"
}

func (p *providerBlock) CompletionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	if p.sr == nil {
		return nil, &noSchemaReaderErr{p.BlockType()}
	}

	cb := &completableBlock{
		logger: p.logger,
	}

	var schemaBlock *tfjson.SchemaBlock
	if p.RawName() != "" {
		pSchema, err := p.sr.ProviderConfigSchema(p.RawName())
		if err != nil {
			return nil, err
		}
		schemaBlock = pSchema.Block
	}
	cb.block = ParseBlock(p.hclBlock, p.Labels(), schemaBlock)

	return cb.completionCandidatesAtPos(pos)
}
