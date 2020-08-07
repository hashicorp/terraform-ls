package lang

import (
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/terraform/addrs"
)

func parseProviderRef(attrs hclsyntax.Attributes, bType string) (addrs.LocalProviderConfig, error) {
	attr, defined := attrs["provider"]
	if !defined {
		// If provider _isn't_ set then we'll infer it from the type.
		return addrs.LocalProviderConfig{
			LocalName: defaultProviderNameFromType(bType),
		}, nil
	}

	// New style here is to provide this as a naked traversal
	// expression, but we also support quoted references for
	// older configurations that predated this convention.
	traversal, travDiags := hcl.AbsTraversalForExpr(attr.Expr)
	if travDiags.HasErrors() {
		traversal = nil // in case we got any partial results

		// Fall back on trying to parse as a string
		var travStr string
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &travStr)
		if !valDiags.HasErrors() {
			var strDiags hcl.Diagnostics
			traversal, strDiags = hclsyntax.ParseTraversalAbs([]byte(travStr), "", hcl.Pos{})
			if strDiags.HasErrors() {
				traversal = nil
			}
		}
	}

	return addrs.ParseProviderConfigCompact(traversal)
}

func defaultProviderNameFromType(typeName string) string {
	if underPos := strings.IndexByte(typeName, '_'); underPos != -1 {
		return typeName[:underPos]
	}
	return typeName
}

func lookupProviderAddress(refs addrs.ProviderReferences, ref addrs.LocalProviderConfig) (addrs.Provider, error) {
	if ref.LocalName == "" {
		return addrs.Provider{}, &UnknownProviderErr{}
	}
	for localRef, pAddr := range refs {
		if localRef.LocalName == ref.LocalName {
			return pAddr, nil
		}
	}

	return addrs.ImpliedProviderForUnqualifiedType(ref.LocalName), nil
}
