// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type StackBlockValidName struct{}

func (sb StackBlockValidName) Visit(ctx context.Context, node hclsyntax.Node, nodeSchema schema.Schema) (context.Context, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	block, ok := node.(*hclsyntax.Block)
	if !ok {
		return ctx, diags
	}

	if nodeSchema == nil {
		return ctx, diags
	}

	// TODO: Revist checking Stack concepts in the static validators
	// 	Other validators are more generic and can be applied to any block/attribute/etc, but
	// 	this one is specific to the stack block types. I'm sure this could be done in a more generic way,
	// 	but I'm not sure if it's worth it at the moment. The names of these blocks are used as identifiers
	//  in the stack and are used in the UI, so they have to be valid Terraform identifiers or else the stack
	//  will not be able to be created. This is a very basic check to ensure that the names are valid until we can
	//  do more advanced checks or use the cloud api.
	//  I tried adding this to the earlydecoder, but we do not seem to report the diagnostics there at all currently,
	//  so nothing was being reported. I'm not sure if that's a bug or intended.

	switch block.Type {
	// stack
	case "component":
		diags = hasValidNameLabel(block, diags)
	// case "provider":
	// case "required_providers":
	case "variable":
		diags = hasValidNameLabel(block, diags)
	case "output":
		diags = hasValidNameLabel(block, diags)
	// deployment
	case "deployment":
		diags = hasValidNameLabel(block, diags)
	case "identity_token":
		diags = hasValidNameLabel(block, diags)
	case "orchestrate":
		diags = hasValidNameLabel(block, diags)
	}

	return ctx, diags
}

func hasValidNameLabel(block *hclsyntax.Block, diags hcl.Diagnostics) hcl.Diagnostics {
	if len(block.Labels) > 0 && !hclsyntax.ValidIdentifier(block.Labels[0]) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invalid %q name %q", block.Type, block.Labels[0]),
			Detail:   "Names must be valid identifiers: beginning with a letter or underscore, followed by zero or more letters, digits, or underscores.",
			Subject:  block.LabelRanges[0].Ptr(),
		})
	}
	return diags
}
