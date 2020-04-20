package lang

import (
	"fmt"
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	lsp "github.com/sourcegraph/go-lsp"
)

type configBlockFactory interface {
	New(*hclsyntax.Block) (ConfigBlock, error)
}

type completableBlock struct {
	logger   *log.Logger
	caps     lsp.TextDocumentClientCapabilities
	hclBlock *hclsyntax.Block
	schema   *tfjson.SchemaBlock
}

func (cb *completableBlock) completionItemsAtPos(pos hcl.Pos) (lsp.CompletionList, error) {
	list := lsp.CompletionList{}

	cb.logger.Printf("block: %#v", cb.hclBlock)

	block := ParseBlock(cb.hclBlock, cb.schema)

	if !block.PosInBody(pos) {
		// Avoid autocompleting outside of body, for now
		cb.logger.Println("avoiding completion outside of block body")
		return list, nil
	}

	if block.PosInAttribute(pos) {
		cb.logger.Println("avoiding completion in the middle of existing attribute")
		return list, nil
	}

	b, ok := block.BlockAtPos(pos)
	if !ok {
		// This should never happen as the completion
		// should only be called on a block the "pos" points to
		cb.logger.Printf("block type not found at %#v", pos)
		return list, nil
	}

	for name, attr := range b.Attributes() {
		if attr.IsComputedOnly() || attr.IsDeclared() {
			continue
		}
		list.Items = append(list.Items, cb.completionItemForAttr(name, attr, pos))
	}

	for name, block := range b.BlockTypes() {
		if block.ReachedMaxItems() {
			continue
		}
		list.Items = append(list.Items, cb.completionItemForNestedBlock(name, block, pos))
	}

	sortCompletionItems(list.Items)

	return list, nil
}

func (cb *completableBlock) completionItemForAttr(name string, attr *Attribute, pos hcl.Pos) lsp.CompletionItem {
	snippetSupport := cb.caps.Completion.CompletionItem.SnippetSupport

	if snippetSupport {
		return lsp.CompletionItem{
			Label:            name,
			Kind:             lsp.CIKField,
			InsertTextFormat: lsp.ITFSnippet,
			Detail:           schemaAttributeDetail(attr.Schema()),
			TextEdit: &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
					End:   lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
				},
				NewText: fmt.Sprintf("%s = %s", name, snippetForAttrType(0, attr.Schema().AttributeType)),
			},
		}
	}

	return lsp.CompletionItem{
		Label:            name,
		Kind:             lsp.CIKField,
		InsertTextFormat: lsp.ITFPlainText,
		Detail:           schemaAttributeDetail(attr.Schema()),
	}
}

func (cb *completableBlock) completionItemForNestedBlock(name string, blockType *BlockType, pos hcl.Pos) lsp.CompletionItem {
	snippetSupport := cb.caps.Completion.CompletionItem.SnippetSupport

	if snippetSupport {
		return lsp.CompletionItem{
			Label:            name,
			Kind:             lsp.CIKField,
			InsertTextFormat: lsp.ITFSnippet,
			Detail:           schemaBlockDetail(blockType),
			TextEdit: &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
					End:   lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1},
				},
				NewText: snippetForNestedBlock(name),
			},
		}
	}

	return lsp.CompletionItem{
		Label:            name,
		Kind:             lsp.CIKField,
		InsertTextFormat: lsp.ITFPlainText,
		Detail:           schemaBlockDetail(blockType),
	}
}
