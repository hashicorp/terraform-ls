package lang

import (
	"fmt"
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type resourceBlockFactory struct {
	logger *log.Logger

	schemaReader schema.Reader
}

func (f *resourceBlockFactory) New(tBlock ihcl.TokenizedBlock) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	return &resourceBlock{
		logger: f.logger,

		labelSchema: f.LabelSchema(),
		tBlock:      tBlock,
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
	tBlock      ihcl.TokenizedBlock
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
	labels := ParseLabels(r.tBlock, r.labelSchema)
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

	hclBlock, _ := hclsyntax.ParseBlockFromTokens(r.tBlock.Tokens())
	if PosInLabels(hclBlock, pos) {
		resources, err := r.sr.Resources()
		if err != nil {
			return nil, err
		}
		cl := &completableLabels{
			logger:       r.logger,
			parsedLabels: r.Labels(),
			tBlock:       r.tBlock,
			labels: labelCandidates{
				"type": resourceCandidates(resources),
			},
		}

		return cl.completionCandidatesAtPos(pos)
	}

	rSchema, err := r.sr.ResourceSchema(r.Type())
	if err != nil {
		return nil, err
	}
	cb := &completableBlock{
		logger:       r.logger,
		parsedLabels: r.Labels(),
		schema:       rSchema.Block,
		tBlock:       r.tBlock,
	}
	return cb.completionCandidatesAtPos(pos)
}

func resourceCandidates(resources []schema.Resource) []*labelCandidate {
	candidates := []*labelCandidate{}
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
