package lang

import (
	"fmt"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	lsp "github.com/sourcegraph/go-lsp"
	"github.com/zclconf/go-cty/cty"
)

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
