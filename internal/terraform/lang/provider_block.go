package lang

import (
	"fmt"
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

	labels := block.Labels
	if len(labels) != 1 {
		return nil, &invalidLabelsErr{f.BlockType(), labels}
	}

	return &providerBlock{
		hclBlock: block,
		logger:   f.logger,
		caps:     f.caps,
		sr:       f.schemaReader,
	}, nil
}

func (f *providerBlockFactory) BlockType() string {
	return "provider"
}

type providerBlock struct {
	logger   *log.Logger
	caps     lsp.TextDocumentClientCapabilities
	hclBlock *hclsyntax.Block
	sr       schema.Reader
}

func (p *providerBlock) Name() string {
	return p.hclBlock.Labels[0]
}

func (p *providerBlock) BlockType() string {
	return "provider"
}

func (p *providerBlock) CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error) {
	list := lsp.CompletionList{}

	if p.sr == nil {
		return list, &noSchemaReaderErr{p.BlockType()}
	}

	pSchema, err := p.sr.ProviderConfigSchema(p.Name())
	if err != nil {
		return list, err
	}

	block := ParseBlock(p.hclBlock, pSchema.Block)

	if !block.PosInBody(pos) {
		// Avoid autocompleting outside of body, for now
		p.logger.Println("avoiding completion outside of block body")
		return list, nil
	}

	if block.PosInAttribute(pos) {
		p.logger.Println("avoiding completion in the middle of existing attribute")
		return list, nil
	}

	b, ok := block.BlockAtPos(pos)
	if !ok {
		// This should never happen as the completion
		// should only be called on a block the "pos" points to
		p.logger.Printf("block type not found at %#v", pos)
		return list, nil
	}

	for name, attr := range b.Attributes() {
		if attr.IsComputedOnly() || attr.IsDeclared() {
			continue
		}
		list.Items = append(list.Items, p.completionItemForAttr(name, attr.Schema(), pos))
	}

	for name, block := range b.BlockTypes() {
		if block.ReachedMaxItems() {
			continue
		}
		list.Items = append(list.Items, p.completionItemForNestedBlock(name, block, pos))
	}

	sortCompletionItems(list.Items)

	return list, nil
}

func (p *providerBlock) completionItemForAttr(name string, sAttr *tfjson.SchemaAttribute,
	pos hcl.Pos) lsp.CompletionItem {

	snippetSupport := p.caps.Completion.CompletionItem.SnippetSupport

	if snippetSupport {
		return lsp.CompletionItem{
			Label:            name,
			Kind:             lsp.CIKField,
			InsertTextFormat: lsp.ITFSnippet,
			Detail:           schemaAttributeDetail(sAttr),
			TextEdit: &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
					End:   lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
				},
				NewText: fmt.Sprintf("%s = %s", name, snippetForAttr(sAttr)),
			},
		}
	}

	return lsp.CompletionItem{
		Label:            name,
		Kind:             lsp.CIKField,
		InsertTextFormat: lsp.ITFPlainText,
		Detail:           schemaAttributeDetail(sAttr),
	}
}

func (p *providerBlock) completionItemForNestedBlock(name string, blockType *BlockType, pos hcl.Pos) lsp.CompletionItem {

	// snippetSupport := p.caps.Completion.CompletionItem.SnippetSupport

	return lsp.CompletionItem{
		Label:            name,
		Kind:             lsp.CIKField,
		InsertTextFormat: lsp.ITFPlainText,
		Detail:           schemaBlockDetail(blockType),
	}
}
