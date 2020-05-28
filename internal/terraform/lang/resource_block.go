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

func (f *resourceBlockFactory) New(tokens hclsyntax.Tokens) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	return &resourceBlock{
		logger: f.logger,

		labelSchema: f.LabelSchema(),
		tokens:      tokens,
		sr:          f.schemaReader,
	}, nil
}

func (f *resourceBlockFactory) LabelSchema() LabelSchema {
	return LabelSchema{
		Label{Name: "type", IsCompletable: true},
		Label{Name: "name", IsCompletable: false},
	}
}

func (r *resourceBlockFactory) BlockType() string {
	return "resource"
}

func (r *resourceBlockFactory) Documentation() MarkupContent {
	return PlainText("A resource block declares a resource of a given type with a given local name. The name is " +
		"used to refer to this resource from elsewhere in the same Terraform module, but has no significance " +
		"outside of the scope of a module.")
}

type resourceBlock struct {
	logger *log.Logger

	labelSchema LabelSchema
	labels      []*ParsedLabel
	tokens      hclsyntax.Tokens
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

func (r *resourceBlock) Labels() []*ParsedLabel {
	if r.labels != nil {
		return r.labels
	}
	labels, _ := ParseLabels(r.tokens, r.labelSchema)
	r.labels = labels

	return r.labels
}

func (r *resourceBlock) BlockType() string {
	return "resource"
}

func (r *resourceBlock) CompletionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	if r.sr == nil {
		return nil, &noSchemaReaderErr{r.BlockType()}
	}

	var schemaBlock *tfjson.SchemaBlock
	if r.Type() != "" {
		rSchema, err := r.sr.ResourceSchema(r.Type())
		if err != nil {
			return nil, err
		}
		schemaBlock = rSchema.Block
	}
	block, err := ParseBlock(r.tokens, r.Labels(), schemaBlock)
	if err != nil {
		return nil, err
	}

	if block.PosInLabels(pos) {
		resources, err := r.sr.Resources()
		if err != nil {
			return nil, err
		}
		cl := &completableLabels{
			logger: r.logger,
			block:  block,
			labels: labelCandidates{
				"type": resourceCandidates(resources),
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

func resourceCandidates(resources []schema.Resource) []CompletionCandidate {
	candidates := []CompletionCandidate{}
	for _, r := range resources {
		var desc MarkupContent = PlainText(r.Description)
		if r.DescriptionKind == tfjson.SchemaDescriptionKindMarkdown {
			desc = Markdown(r.Description)
		}

		candidates = append(candidates, &labelCandidate{
			label:         r.Name,
			detail:        fmt.Sprintf("Resource (%s)", r.Provider),
			documentation: desc,
		})
	}
	return candidates
}
