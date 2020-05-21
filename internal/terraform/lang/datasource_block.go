package lang

import (
	"fmt"
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type datasourceBlockFactory struct {
	logger *log.Logger

	schemaReader schema.Reader
}

func (f *datasourceBlockFactory) New(block *hclsyntax.Block) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	return &datasourceBlock{
		logger: f.logger,

		labelSchema: f.LabelSchema(),
		hclBlock:    block,
		sr:          f.schemaReader,
	}, nil
}

func (f *datasourceBlockFactory) LabelSchema() LabelSchema {
	return LabelSchema{
		Label{Name: "type", IsCompletable: true},
		Label{Name: "name"},
	}
}

func (f *datasourceBlockFactory) BlockType() string {
	return "data"
}

type datasourceBlock struct {
	logger *log.Logger

	labelSchema LabelSchema
	labels      []*ParsedLabel
	hclBlock    *hclsyntax.Block
	sr          schema.Reader
}

func (r *datasourceBlock) Type() string {
	return r.Labels()[0].Value
}

func (r *datasourceBlock) Name() string {
	firstLabel := r.Labels()[0].Value
	secondLabel := r.Labels()[1].Value

	if firstLabel == "" && secondLabel == "" {
		return "<unknown>"
	}
	if firstLabel == "" {
		firstLabel = "<unknown>"
	}
	if secondLabel == "" {
		secondLabel = "<unknown>"
	}

	return fmt.Sprintf("%s.%s", firstLabel, secondLabel)
}

func (r *datasourceBlock) Labels() []*ParsedLabel {
	if r.labels != nil {
		return r.labels
	}
	r.labels = parseLabels(r.BlockType(), r.labelSchema, r.hclBlock.Labels)
	return r.labels
}

func (r *datasourceBlock) BlockType() string {
	return "data"
}

func (r *datasourceBlock) CompletionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	if r.sr == nil {
		return nil, &noSchemaReaderErr{r.BlockType()}
	}

	var schemaBlock *tfjson.SchemaBlock
	if r.Type() != "" {
		rSchema, err := r.sr.DataSourceSchema(r.Type())
		if err != nil {
			return nil, err
		}
		schemaBlock = rSchema.Block
	}
	block := ParseBlock(r.hclBlock, r.Labels(), schemaBlock)

	if block.PosInLabels(pos) {
		dataSources, err := r.sr.DataSources()
		if err != nil {
			return nil, err
		}

		cl := &completableLabels{
			logger: r.logger,
			block:  block,
			labels: labelCandidates{
				"type": dataSourceCandidates(dataSources),
			},
		}

		return cl.completionCandidatesAtPos(pos)
	}

	cb := &completableBlock{
		logger: r.logger,
		block:  block,
	}
	return cb.completionCandidatesAtPos(pos)
}

func dataSourceCandidates(dataSources []schema.DataSource) []CompletionCandidate {
	candidates := []CompletionCandidate{}
	for _, ds := range dataSources {
		candidates = append(candidates, &labelCandidate{
			label:  ds.Name,
			detail: ds.Provider,
		})
	}
	return candidates
}
