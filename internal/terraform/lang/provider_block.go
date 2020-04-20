package lang

import (
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	lsp "github.com/sourcegraph/go-lsp"
)

type providerBlockFactory struct {
	logger *log.Logger
	caps   lsp.TextDocumentClientCapabilities

	schemaReader schema.Reader
}

func (f *providerBlockFactory) New(block *hclsyntax.Block) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	return &providerBlock{
		logger: f.logger,
		caps:   f.caps,

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
	caps   lsp.TextDocumentClientCapabilities

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

func (p *providerBlock) CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error) {
	list := lsp.CompletionList{}

	if p.sr == nil {
		return list, &noSchemaReaderErr{p.BlockType()}
	}

	cb := &completableBlock{
		logger: p.logger,
		caps:   p.caps,
	}

	var schemaBlock *tfjson.SchemaBlock
	if p.RawName() != "" {
		pSchema, err := p.sr.ProviderConfigSchema(p.RawName())
		if err != nil {
			return list, err
		}
		schemaBlock = pSchema.Block
	}
	cb.block = ParseBlock(p.hclBlock, p.Labels(), schemaBlock)

	return cb.completionItemsAtPos(pos)
}
