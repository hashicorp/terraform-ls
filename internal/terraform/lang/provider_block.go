package lang

import (
	"fmt"
	"log"
	"sort"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	lsp "github.com/sourcegraph/go-lsp"
	"github.com/zclconf/go-cty/cty"
)

type providerBlockFactory struct {
	logger *log.Logger
	caps   lsp.TextDocumentClientCapabilities
}

func (f *providerBlockFactory) New(block *hcl.Block) (ConfigBlock, error) {
	if f.logger == nil {
		f.logger = emptyLogger()
	}
	if block == nil {
		return nil, EmptyConfigErr()
	}
	labels := block.Labels
	if len(labels) != 1 {
		return nil, &InvalidLabelsErr{"provider", labels}
	}

	return &providerBlock{hclBlock: block, logger: f.logger, caps: f.caps}, nil
}

func (f *providerBlockFactory) BlockType() string {
	return "provider"
}

func (f *providerBlockFactory) InitializeCapabilities(caps lsp.TextDocumentClientCapabilities) {
	f.caps = caps
}

type providerBlock struct {
	logger   *log.Logger
	caps     lsp.TextDocumentClientCapabilities
	hclBlock *hcl.Block
	schema   *tfjson.Schema
}

func (p *providerBlock) CompletionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error) {
	list := lsp.CompletionList{}

	if p.schema == nil {
		return list, &SchemaUnavailableErr{"provider", p.Name()}
	}

	hs := jsonSchemaToHcl(p.schema)

	content, body, diags := p.hclBlock.Body.PartialContent(hs)
	if diags.HasErrors() {
		p.logger.Printf("mapping schema to config tolerated errors: %s", diags)
	}

	hclBody, ok := body.(*hclsyntax.Body)
	if !ok {
		// if user happens to be editing JSON
		return list, fmt.Errorf("unsupported body type: %T", body)
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

	attrs := undeclaredSchemaAttributes(p.schema.Block.Attributes, content.Attributes)
	// TODO: blocks := undeclaredSchemaBlocks(p.schema.Block.NestedBlocks, content.Blocks)

	for name, attr := range attrs {
		if attr.Computed && !attr.Optional && !attr.Required {
			continue
		}

		list.Items = append(list.Items, p.completionItem(name, attr, pos))
	}

	sortCompletionItems(list.Items)

	return list, nil
}

func sortCompletionItems(items []lsp.CompletionItem) {
	less := func(i, j int) bool {
		return items[i].Label < items[j].Label
	}
	sort.Slice(items, less)
}

func (p *providerBlock) completionItem(name string, attr *tfjson.SchemaAttribute,
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

func snippetForAttr(attr *tfjson.SchemaAttribute) string {
	switch attr.AttributeType {
	case cty.String:
		return `"${0:value}"`
	case cty.Bool:
		return `${0:false}`
	case cty.Number:
		return `${0:42}`
	}
	return ""
}

func schemaAttributeDetail(attr *tfjson.SchemaAttribute) string {
	var requiredText string
	if attr.Optional {
		requiredText = "Optional"
	}
	if attr.Required {
		requiredText = "Required"
	}

	return strings.TrimSpace(fmt.Sprintf("(%s, %s) %s",
		requiredText, attr.AttributeType.FriendlyName(), attr.Description))
}

func (p *providerBlock) Name() string {
	return p.hclBlock.Labels[0]
}

func (p *providerBlock) BlockType() string {
	return "provider"
}

func (p *providerBlock) LoadSchema(ps *tfjson.ProviderSchemas) error {
	providerName := p.Name()

	schema, ok := ps.Schemas[providerName]
	if !ok {
		return &SchemaUnavailableErr{"provider", providerName}
	}

	p.schema = schema.ConfigSchema
	return nil
}
