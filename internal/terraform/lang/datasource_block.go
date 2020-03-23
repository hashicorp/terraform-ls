package lang

import (
	"log"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	lsp "github.com/sourcegraph/go-lsp"
)

type datasourceBlockFactory struct {
	logger *log.Logger
	caps   lsp.TextDocumentClientCapabilities

	schemaReader schema.Reader
}

func (f *datasourceBlockFactory) New(block *hclsyntax.Block) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	labels := block.Labels
	if len(labels) != 2 {
		return nil, &invalidLabelsErr{f.BlockType(), labels}
	}

	return &datasourceBlock{
		hclBlock: block,
		logger:   f.logger,
		caps:     f.caps,
		sr:       f.schemaReader,
	}, nil
}

func (r *datasourceBlockFactory) BlockType() string {
	return "data"
}

type datasourceBlock struct {
	logger   *log.Logger
	caps     lsp.TextDocumentClientCapabilities
	hclBlock *hclsyntax.Block
	sr       schema.Reader
}

func (r *datasourceBlock) Type() string {
	return r.hclBlock.Labels[0]
}

func (r *datasourceBlock) Name() string {
	return strings.Join(r.hclBlock.Labels, ".")
}

func (r *datasourceBlock) BlockType() string {
	return "data"
}

func (r *datasourceBlock) CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error) {
	list := lsp.CompletionList{}

	if r.sr == nil {
		return list, &noSchemaReaderErr{r.BlockType()}
	}

	rSchema, err := r.sr.DataSourceSchema(r.Type())
	if err != nil {
		return list, err
	}

	cb := &completableBlock{
		logger:   r.logger,
		caps:     r.caps,
		hclBlock: r.hclBlock,
		schema:   rSchema.Block,
	}

	return cb.completionItemsAtPos(pos)
}
