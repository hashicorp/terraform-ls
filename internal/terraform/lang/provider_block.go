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

func (f *providerBlockFactory) New(block *hcl.Block) (ConfigBlock, error) {
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
	hclBlock *hcl.Block
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

	hs := jsonSchemaToHcl(pSchema)

	content, body, diags := p.hclBlock.Body.PartialContent(hs)
	if diags.HasErrors() {
		p.logger.Printf("mapping schema to config tolerated errors: %s", diags)
	}

	hclBody, ok := body.(*hclsyntax.Body)
	if !ok {
		return list, &unsupportedConfigTypeErr{body}
	}

	if !bodyContainsPos(hclBody, pos) {
		// Avoid autocompleting outside of body, for now
		p.logger.Println("avoiding completion outside of block body")
		return list, nil
	}

	if contentContainPos(hclBody, pos) {
		// No auto-completing in the middle of existing fields
		p.logger.Println("avoiding completion in the middle of existing field")
		return list, nil
	}

	attrs := undeclaredSchemaAttributes(pSchema.Block.Attributes, content.Attributes)
	// TODO: blocks := undeclaredSchemaBlocks(pSchema.Block.NestedBlocks, content.Blocks)

	for name, attr := range attrs {
		if attr.Computed && !attr.Optional && !attr.Required {
			continue
		}

		list.Items = append(list.Items, p.completionItemForAttr(name, attr, pos))
	}

	sortCompletionItems(list.Items)

	return list, nil
}

func (p *providerBlock) completionItemForAttr(name string, attr *tfjson.SchemaAttribute,
	pos hcl.Pos) lsp.CompletionItem {

	snippetSupport := p.caps.Completion.CompletionItem.SnippetSupport

	if snippetSupport {
		return lsp.CompletionItem{
			Label:            name,
			Kind:             lsp.CIKField,
			InsertTextFormat: lsp.ITFSnippet,
			Detail:           schemaAttributeDetail(attr),
			TextEdit: &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
					End:   lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
				},
				NewText: fmt.Sprintf("%s = %s", name, snippetForAttr(attr)),
			},
		}
	}

	return lsp.CompletionItem{
		Label:            name,
		Kind:             lsp.CIKField,
		InsertTextFormat: lsp.ITFPlainText,
		Detail:           schemaAttributeDetail(attr),
	}
}
