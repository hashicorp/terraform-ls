package lang

import (
	"log"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	lsp "github.com/sourcegraph/go-lsp"
)

type resourceBlockFactory struct {
	logger *log.Logger
	caps   lsp.TextDocumentClientCapabilities

	schemaReader schema.Reader
}

func (f *resourceBlockFactory) New(block *hclsyntax.Block) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = discardLog()
	}

	labels := block.Labels
	if len(labels) != 2 {
		return nil, &invalidLabelsErr{f.BlockType(), labels}
	}

	return &resourceBlock{
		hclBlock: block,
		logger:   f.logger,
		caps:     f.caps,
		sr:       f.schemaReader,
	}, nil
}

func (r *resourceBlockFactory) BlockType() string {
	return "resource"
}

type resourceBlock struct {
	logger   *log.Logger
	caps     lsp.TextDocumentClientCapabilities
	hclBlock *hclsyntax.Block
	sr       schema.Reader
}

func (r *resourceBlock) Type() string {
	return r.hclBlock.Labels[0]
}

func (r *resourceBlock) Name() string {
	return strings.Join(r.hclBlock.Labels, ".")
}

func (r *resourceBlock) BlockType() string {
	return "resource"
}

func (r *resourceBlock) CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error) {
	list := lsp.CompletionList{}

	if r.sr == nil {
		return list, &noSchemaReaderErr{r.BlockType()}
	}

	rSchema, err := r.sr.ResourceSchema(r.Type())
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
