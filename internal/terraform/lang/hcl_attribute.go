package lang

import (
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
)

type Attribute struct {
	schema       *tfjson.SchemaAttribute
	hclAttribute *hcl.Attribute
}

func (a *Attribute) Schema() *tfjson.SchemaAttribute {
	return a.schema
}

func (a *Attribute) Range() hcl.Range {
	return a.hclAttribute.Range
}

func (a *Attribute) IsDeclared() bool {
	return a.hclAttribute != nil
}

func (a *Attribute) IsComputedOnly() bool {
	s := a.schema
	return s.Computed && !s.Optional && !s.Required
}

func parseAttributes(hclAttrs hclsyntax.Attributes, schemaAttrs map[string]*tfjson.SchemaAttribute) (
	map[string]*Attribute, hclsyntax.Attributes) {

	attributes := make(map[string]*Attribute, 0)

	for name, attrS := range schemaAttrs {
		attr, declared := hclAttrs[name]
		a := &Attribute{
			schema: attrS,
		}
		if declared {
			a.hclAttribute = attr.AsHCLAttribute()
		}

		attributes[name] = a
		delete(hclAttrs, name)
	}

	return attributes, hclAttrs
}
