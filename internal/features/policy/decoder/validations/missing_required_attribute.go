// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl-lang/schemacontext"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type MissingRequiredAttribute struct{}

func (mra MissingRequiredAttribute) Visit(ctx context.Context, node hclsyntax.Node, nodeSchema schema.Schema) (context.Context, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	if HasUnknownRequiredAttributes(ctx) {
		return ctx, diags
	}

	switch nodeType := node.(type) {
	case *hclsyntax.Block:
		// Providers are excluded from the validation for the time being
		// due to complexity around required attributes with dynamic defaults
		// See https://github.com/hashicorp/vscode-terraform/issues/1616
		nestingLvl, nestingOk := schemacontext.BlockNestingLevel(ctx)
		if nodeType.Type == "provider" && (nestingOk && nestingLvl == 0) {
			ctx = WithUnknownRequiredAttributes(ctx)
		}
	case *hclsyntax.Body:
		if nodeSchema == nil {
			return ctx, diags
		}

		bodySchema := nodeSchema.(*schema.BodySchema)
		if bodySchema.Attributes == nil {
			return ctx, diags
		}

		for name, attr := range bodySchema.Attributes {
			if attr.IsRequired {
				_, ok := nodeType.Attributes[name]
				if !ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  fmt.Sprintf("Required attribute %q not specified", name),
						Detail:   fmt.Sprintf("An attribute named %q is required here", name),
						Subject:  nodeType.SrcRange.Ptr(),
					})
				}
			}
		}
	}

	return ctx, diags
}

type unknownRequiredAttrsCtxKey struct{}

func HasUnknownRequiredAttributes(ctx context.Context) bool {
	_, ok := ctx.Value(unknownRequiredAttrsCtxKey{}).(bool)
	return ok
}

func WithUnknownRequiredAttributes(ctx context.Context) context.Context {
	return context.WithValue(ctx, unknownRequiredAttrsCtxKey{}, true)
}
