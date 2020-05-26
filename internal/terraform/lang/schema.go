package lang

import (
	"fmt"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

func snippetForAttrType(placeholder int, attrType cty.Type) string {
	mapSnippet := func(aType cty.Type) string {
		return fmt.Sprintf("{\n"+`  "${0:key}" = %s`+"\n}",
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

func snippetForBlock(name string, labelSchema LabelSchema) string {
	bodyPlaceholder := 0
	labels := make([]string, len(labelSchema))
	for i, l := range labelSchema {
		if l.IsCompletable {
			labels[i] = fmt.Sprintf(`"${%d}"`, i+1)
		} else {
			labels[i] = fmt.Sprintf(`"${%d:%s}"`, i+1, l.Name)
		}
		bodyPlaceholder = i + 2
	}

	return fmt.Sprintf("%s %s {\n  ${%d}\n}",
		name, strings.Join(labels, " "), bodyPlaceholder)
}

func schemaAttributeDetail(attr *tfjson.SchemaAttribute) string {
	var requiredText string
	if attr.Optional {
		requiredText = "Optional"
	}
	if attr.Required {
		requiredText = "Required"
	}

	return strings.TrimSpace(fmt.Sprintf("%s, %s",
		requiredText, attr.AttributeType.FriendlyName()))
}

func schemaBlockDetail(blockType *BlockType) string {
	blockS := blockType.Schema()

	detail := fmt.Sprintf("Block, %s", blockS.NestingMode)

	if blockS.MinItems > 0 {
		detail += fmt.Sprintf(", min: %d", blockS.MinItems)
	}
	if blockS.MaxItems > 0 {
		detail += fmt.Sprintf(", max: %d", blockS.MaxItems)
	}

	return strings.TrimSpace(detail)
}
