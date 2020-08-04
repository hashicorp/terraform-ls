package lang

import (
	"fmt"
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
	"github.com/hashicorp/terraform-ls/internal/terraform/addrs"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type providerBlockFactory struct {
	logger *log.Logger

	schemaReader schema.Reader
}

func (f *providerBlockFactory) New(tBlock ihcl.TokenizedBlock) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	return &providerBlock{
		logger: f.logger,

		labelSchema: f.LabelSchema(),
		tBlock:      tBlock,
		sr:          f.schemaReader,
	}, nil
}

func (f *providerBlockFactory) LabelSchema() LabelSchema {
	return LabelSchema{
		Label{Name: "name", IsCompletable: true},
	}
}

func (f *providerBlockFactory) BlockType() string {
	return "provider"
}

func (f *providerBlockFactory) Documentation() MarkupContent {
	return PlainText("A provider block is used to specify a provider configuration. The body of the block (between " +
		"{ and }) contains configuration arguments for the provider itself. Most arguments in this section are " +
		"specified by the provider itself.")
}

type providerBlock struct {
	logger *log.Logger

	labelSchema LabelSchema
	labels      []*ParsedLabel
	tBlock      ihcl.TokenizedBlock
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

func (p *providerBlock) Labels() []*ParsedLabel {
	if p.labels != nil {
		return p.labels
	}

	labels := ParseLabels(p.tBlock, p.labelSchema)
	p.labels = labels

	return p.labels
}

func (p *providerBlock) BlockType() string {
	return "provider"
}

func (p *providerBlock) CompletionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	if p.sr == nil {
		return nil, &noSchemaReaderErr{p.BlockType()}
	}

	hclBlock, _ := hclsyntax.ParseBlockFromTokens(p.tBlock.Tokens())
	if PosInLabels(hclBlock, pos) {
		providers, err := p.sr.Providers()
		if err != nil {
			return nil, err
		}

		cl := &completableLabels{
			logger:       p.logger,
			parsedLabels: p.Labels(),
			tBlock:       p.tBlock,
			labels: labelCandidates{
				"name": providerCandidates(providers),
			},
		}

		return cl.completionCandidatesAtPos(pos)
	}

	rawName := p.RawName()
	if rawName == "" {
		return nil, fmt.Errorf("unknown provider name at %#v", pos)
	}
	addr := addrs.ImpliedProviderForUnqualifiedType(rawName)

	pSchema, err := p.sr.ProviderConfigSchema(addr)
	if err != nil {
		return nil, err
	}
	cb := &completableBlock{
		logger: p.logger,
		schema: pSchema.Block,
		tBlock: p.tBlock,
	}
	return cb.completionCandidatesAtPos(pos)
}

func providerCandidates(providers []addrs.Provider) []*labelCandidate {
	candidates := []*labelCandidate{}
	for _, pAddr := range providers {
		candidates = append(candidates, &labelCandidate{
			label:         pAddr.Type,
			detail:        pAddr.ForDisplay(),
			documentation: PlainText(pAddr.Description),
		})
	}
	return candidates
}
