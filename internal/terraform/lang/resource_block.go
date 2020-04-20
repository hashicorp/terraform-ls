package lang

import (
	"fmt"
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type resourceBlockFactory struct {
	logger *log.Logger

	schemaReader schema.Reader
}

func (f *resourceBlockFactory) New(block *hclsyntax.Block) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	return &resourceBlock{
		logger: f.logger,

		labelSchema: LabelSchema{"type", "name"},
		hclBlock:    block,
		sr:          f.schemaReader,
	}, nil
}

func (r *resourceBlockFactory) BlockType() string {
	return "resource"
}

type resourceBlock struct {
	logger *log.Logger

	labelSchema LabelSchema
	labels      []*Label
	hclBlock    *hclsyntax.Block
	sr          schema.Reader
}

func (r *resourceBlock) Type() string {
	return r.Labels()[0].Value
}

func (r *resourceBlock) Name() string {
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

func (r *resourceBlock) Labels() []*Label {
	if r.labels != nil {
		return r.labels
	}
	r.labels = parseLabels(r.BlockType(), r.labelSchema, r.hclBlock.Labels)
	return r.labels
}

func (r *resourceBlock) BlockType() string {
	return "resource"
}

func (r *resourceBlock) CompletionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	if r.sr == nil {
		return nil, &noSchemaReaderErr{r.BlockType()}
	}

	cb := &completableBlock{
		logger: r.logger,
	}

	var schemaBlock *tfjson.SchemaBlock
	if r.Type() != "" {
		rSchema, err := r.sr.ResourceSchema(r.Type())
		if err != nil {
			return nil, err
		}
		schemaBlock = rSchema.Block
	}
	cb.block = ParseBlock(r.hclBlock, r.Labels(), schemaBlock)

	return cb.completionCandidatesAtPos(pos)
}
