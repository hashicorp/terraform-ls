package lang

import (
	"fmt"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	lsp "github.com/sourcegraph/go-lsp"
	"github.com/zclconf/go-cty/cty"
)

func snippetForAttrType(placeholder int, attrType cty.Type) string {
	mapSnippet := func(aType cty.Type) string {
		return fmt.Sprintf("{\n"+`  ${0:key} = %s`+"\n}",
			snippetForAttrType(1, aType))
	}

	switch attrType {
	case cty.String:
		return fmt.Sprintf(`"${%d:value}"`, placeholder)
	case cty.List(cty.String), cty.Set(cty.String):
		return fmt.Sprintf(`["${%d:value}"]`, placeholder)
	case cty.Map(cty.String):
		return mapSnippet(cty.String)

	case cty.Bool:
		return fmt.Sprintf(`${%d:false}`, placeholder)
	case cty.List(cty.Bool), cty.Set(cty.Bool):
		return fmt.Sprintf(`[${%d:false}]`, placeholder)
	case cty.Map(cty.Bool):
		return mapSnippet(cty.Bool)

	case cty.Number:
		return fmt.Sprintf(`${%d:42}`, placeholder)
	case cty.List(cty.Number), cty.Set(cty.Number):
		return fmt.Sprintf(`[${%d:42}]`, placeholder)
	case cty.Map(cty.Number):
		return mapSnippet(cty.Number)
	}

	return ""
}

func snippetForNestedBlock(name string) string {
	return fmt.Sprintf("%s {\n  ${0}\n}", name)
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

func schemaBlockDetail(blockType *BlockType) string {
	blockS := blockType.Schema()

	requiredText := "Required"
	if len(blockType.BlockList) >= int(blockS.MinItems) {
		requiredText = "Optional"
	}

	return strings.TrimSpace(fmt.Sprintf("(%s, %s)",
		requiredText, blockS.NestingMode))
}

func sortCompletionItems(items []lsp.CompletionItem) {
	less := func(i, j int) bool {
		return items[i].Label < items[j].Label
	}
	sort.Slice(items, less)
}
