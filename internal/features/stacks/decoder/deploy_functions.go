// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package decoder

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
)

// source: https://github.com/hashicorp/tfc-agent/blob/fb9349a7d09985b206d32b218b36fa9ad2515478/core/components/stacks/lang/functions.go#L17
var deployFunctionNames = []string{
	"abs",
	"can",
	"ceil",
	"chomp",
	"coalescelist",
	"compact",
	"concat",
	"contains",
	"csvdecode",
	"distinct",
	"element",
	"chunklist",
	"flatten",
	"floor",
	"format",
	"formatdate",
	"formatlist",
	"indent",
	"join",
	"jsondecode",
	"jsonencode",
	"keys",
	"log",
	"lower",
	"max",
	"merge",
	"min",
	"parseint",
	"pow",
	"range",
	"regex",
	"regexall",
	"reverse",
	"setintersection",
	"setproduct",
	"setsubtract",
	"setunion",
	"signum",
	"slice",
	"sort",
	"split",
	"strrev",
	"substr",
	"timeadd",
	"title",
	"trim",
	"trimprefix",
	"trimspace",
	"trimsuffix",
	"try",
	"upper",
	"values",
	"yamldecode",
	"yamlencode",
	"zipmap",
}

func deployFunctionsForVersion(v *version.Version) map[string]schema.FunctionSignature {
	fns := mustFunctionsForVersion(v)

	deployFns := make(map[string]schema.FunctionSignature)
	for _, name := range deployFunctionNames {
		if fn, ok := fns[name]; ok {
			deployFns[name] = *fn.Copy()
		}
	}

	return deployFns
}
