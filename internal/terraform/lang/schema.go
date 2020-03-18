package lang

import (
	"fmt"
	"sort"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	lsp "github.com/sourcegraph/go-lsp"
	"github.com/zclconf/go-cty/cty"
)

func undeclaredSchemaAttributes(attrs map[string]*tfjson.SchemaAttribute,
	declared hcl.Attributes) map[string]*tfjson.SchemaAttribute {

	for name, _ := range attrs {
		if _, ok := declared[name]; ok {
			delete(attrs, name)
		}
	}

	return attrs
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

func sortCompletionItems(items []lsp.CompletionItem) {
	less := func(i, j int) bool {
		return items[i].Label < items[j].Label
	}
	sort.Slice(items, less)
}
