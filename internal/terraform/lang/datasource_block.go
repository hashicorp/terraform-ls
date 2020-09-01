package lang

import (
	"fmt"
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
	"github.com/hashicorp/terraform-ls/internal/terraform/addrs"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type datasourceBlockFactory struct {
	logger *log.Logger

	schemaReader schema.Reader
	providerRefs addrs.ProviderReferences
}

func (f *datasourceBlockFactory) New(tBlock ihcl.TokenizedBlock) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	return &datasourceBlock{
		logger: f.logger,

		labelSchema:  f.LabelSchema(),
		tBlock:       tBlock,
		sr:           f.schemaReader,
		providerRefs: f.providerRefs,
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

	labelSchema  LabelSchema
	labels       []*ParsedLabel
	tBlock       ihcl.TokenizedBlock
	sr           schema.Reader
	providerRefs addrs.ProviderReferences
}

func (d *datasourceBlock) Type() string {
	return d.Labels()[0].Value
}

func (d *datasourceBlock) Name() string {
	firstLabel := d.Labels()[0].Value
	secondLabel := d.Labels()[1].Value

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

func (d *datasourceBlock) Labels() []*ParsedLabel {
	if d.labels != nil {
		return d.labels
	}

	labels := ParseLabels(d.tBlock, d.labelSchema)
	d.labels = labels

	return d.labels
}

func (d *datasourceBlock) BlockType() string {
	return "data"
}

func (d *datasourceBlock) Range() hcl.Range {
	return d.tBlock.Range()
}

func (d *datasourceBlock) CompletionCandidatesAtPos(pos hcl.Pos) (CompletionCandidates, error) {
	if d.sr == nil {
		return nil, &noSchemaReaderErr{d.BlockType()}
	}

	// We ignore diags as we assume incomplete (invalid) configuration
	hclBlock, _ := hclsyntax.ParseBlockFromTokens(d.tBlock.Tokens())

	if PosInLabels(hclBlock, pos) {
		dataSources, err := d.sr.DataSources()
		if err != nil {
			return nil, err
		}

		cl := &completableLabels{
			logger:       d.logger,
			parsedLabels: d.Labels(),
			tBlock:       d.tBlock,
			labels: labelCandidates{
				"type": dataSourceCandidates(dataSources),
			},
		}

		return cl.completionCandidatesAtPos(pos)
	}

	lRef, err := parseProviderRef(hclBlock.Body.Attributes, d.Type())
	if err != nil {
		return nil, err
	}

	pAddr, err := lookupProviderAddress(d.providerRefs, lRef)
	if err != nil {
		return nil, err
	}

	rSchema, err := d.sr.DataSourceSchema(pAddr, d.Type())
	if err != nil {
		return nil, err
	}
	cb := &completableBlock{
		logger:       d.logger,
		parsedLabels: d.Labels(),
		schema:       rSchema.Block,
		tBlock:       d.tBlock,
	}
	return cb.completionCandidatesAtPos(pos)
}

func dataSourceCandidates(dataSources []schema.DataSource) []*labelCandidate {
	candidates := []*labelCandidate{}
	for _, ds := range dataSources {
		var desc MarkupContent = PlainText(ds.Description)
		if ds.DescriptionKind == tfjson.SchemaDescriptionKindMarkdown {
			desc = Markdown(ds.Description)
		}

		candidates = append(candidates, &labelCandidate{
			label:         ds.Name,
			detail:        fmt.Sprintf("Data Source (%s)", ds.Provider.ForDisplay()),
			documentation: desc,
		})
	}
	return candidates
}
