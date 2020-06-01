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

func (f *datasourceBlockFactory) New(tokens hclsyntax.Tokens) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	return &datasourceBlock{
		logger: f.logger,

		labelSchema: f.LabelSchema(),
		tokens:      tokens,
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

func (f *datasourceBlockFactory) Documentation() MarkupContent {
	return PlainText("A data block requests that Terraform read from a given data source and export the result " +
		"under the given local name. The name is used to refer to this resource from elsewhere in the same " +
		"Terraform module, but has no significance outside of the scope of a module.")
}

type datasourceBlock struct {
	logger *log.Logger

	labelSchema LabelSchema
	labels      []*ParsedLabel
	tokens      hclsyntax.Tokens
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
	labels := ParseLabels(r.tokens, r.labelSchema)
	r.labels = labels

	return r.labels
}

func (r *datasourceBlock) BlockType() string {
	return "data"
}

func (r *datasourceBlock) CompletionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	if r.sr == nil {
		return nil, &noSchemaReaderErr{r.BlockType()}
	}

	hclBlock, _ := hclsyntax.ParseBlockFromTokens(r.tokens)
	if PosInLabels(hclBlock, pos) {
		dataSources, err := r.sr.DataSources()
		if err != nil {
			return nil, err
		}

		cl := &completableLabels{
			logger: r.logger,
			block:  ParseBlock(hclBlock, r.Labels(), nil),
			tokens: r.tokens,
			labels: labelCandidates{
				"type": dataSourceCandidates(dataSources, pos),
			},
		}

		return cl.completionCandidatesAtPos(pos)
	}

	rSchema, err := r.sr.DataSourceSchema(r.Type())
	if err != nil {
		return nil, err
	}
	cb := &completableBlock{
		logger: r.logger,
		block:  ParseBlock(hclBlock, r.Labels(), rSchema.Block),
		tokens: r.tokens,
	}
	return cb.completionCandidatesAtPos(pos)
}

func dataSourceCandidates(dataSources []schema.DataSource, pos hcl.Pos) []CompletionCandidate {
	candidates := []CompletionCandidate{}
	for _, ds := range dataSources {
		var desc MarkupContent = PlainText(ds.Description)
		if ds.DescriptionKind == tfjson.SchemaDescriptionKindMarkdown {
			desc = Markdown(ds.Description)
		}

		candidates = append(candidates, &labelCandidate{
			label:         ds.Name,
			detail:        fmt.Sprintf("Data Source (%s)", ds.Provider),
			documentation: desc,
			pos:           pos,
		})
	}
	return candidates
}
