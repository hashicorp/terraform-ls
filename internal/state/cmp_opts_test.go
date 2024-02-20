// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

var cmpOpts = cmp.Options{
	cmp.AllowUnexported(Module{}),
	cmp.AllowUnexported(datadir.ModuleManifest{}),
	cmp.AllowUnexported(hclsyntax.Body{}),
	cmp.Comparer(func(x, y version.Constraint) bool {
		return x.String() == y.String()
	}),
	cmp.Comparer(func(x, y hcl.File) bool {
		return (x.Body == y.Body &&
			cmp.Equal(x.Bytes, y.Bytes))
	}),
	ctydebug.CmpOptions,
}
