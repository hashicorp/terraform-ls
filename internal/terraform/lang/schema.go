package lang

import (
	"fmt"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

func snippetForAttrType(attrType cty.Type) string {
	text, _ := snippetForAttrTypeWithPlaceholder(1, attrType)
	return text
}

func snippetForAttrTypeWithPlaceholder(placeholder int, attrType cty.Type) (string, int) {
	mapSnippet := func(placeholder int, aType cty.Type) (string, int) {
		text, nextPlaceHolder := snippetForAttrTypeWithPlaceholder(placeholder+1, aType)
		return fmt.Sprintf("{\n"+`  "${%d:key}" = %s`+"\n}",
			placeholder, text), nextPlaceHolder
	}

	nextPlaceHolder := placeholder + 1
	switch attrType {
	case cty.String:
		return fmt.Sprintf(`"${%d:value}"`, placeholder), nextPlaceHolder
	case cty.List(cty.String), cty.Set(cty.String):
		return fmt.Sprintf(`["${%d:value}"]`, placeholder), nextPlaceHolder
	case cty.Map(cty.String):
		return mapSnippet(placeholder, cty.String)

	case cty.Bool:
		return fmt.Sprintf(`${%d:false}`, placeholder), nextPlaceHolder
	case cty.List(cty.Bool), cty.Set(cty.Bool):
		return fmt.Sprintf(`[${%d:false}]`, placeholder), nextPlaceHolder
	case cty.Map(cty.Bool):
		return mapSnippet(placeholder, cty.Bool)

	case cty.Number:
		return fmt.Sprintf(`${%d:0}`, placeholder), nextPlaceHolder
	case cty.List(cty.Number), cty.Set(cty.Number):
		return fmt.Sprintf(`[${%d:0}]`, placeholder), nextPlaceHolder
	case cty.Map(cty.Number):
		return mapSnippet(placeholder, cty.Number)
	}

	return "", placeholder
}

func plainTextForAttrType(attrType cty.Type) string {
	mapPlainText := func(aType cty.Type) string {
		return fmt.Sprintf("{\n"+`  "key" = %s`+"\n}",
			plainTextForAttrType(aType))
	}

	switch attrType {
	case cty.String:
		return `""`
	case cty.List(cty.String), cty.Set(cty.String):
		return fmt.Sprintf(`[""]`)
	case cty.Map(cty.String):
		return mapPlainText(cty.String)

	case cty.Bool:
		return `false`
	case cty.List(cty.Bool), cty.Set(cty.Bool):
		return `[false]`
	case cty.Map(cty.Bool):
		return plainTextForAttrType(cty.Bool)

	case cty.Number:
		return `0`
	case cty.List(cty.Number), cty.Set(cty.Number):
		return `[0]`
	case cty.Map(cty.Number):
		return plainTextForAttrType(cty.Number)
	}

	return ""
}

func snippetForNestedBlock(name string) string {
	return snippetForNestedBlockWithPlaceholder(1, name)
}

func snippetForNestedBlockWithPlaceholder(placeholder int, name string) string {
	return fmt.Sprintf("%s {\n  ${%d}\n}", name, placeholder)
}

func plainTextForNestedBlock(name string) string {
	return fmt.Sprintf("%s {\n  \n}", name)
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
