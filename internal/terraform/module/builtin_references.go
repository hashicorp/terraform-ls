// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package module

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/zclconf/go-cty/cty"
)

var builtinScopeId = lang.ScopeId("builtin")

func builtinReferences(modPath string) reference.Targets {
	return reference.Targets{
		{
			Addr: lang.Address{
				lang.RootStep{Name: "path"},
				lang.AttrStep{Name: "module"},
			},
			ScopeId: builtinScopeId,
			Type:    cty.String,
			Description: lang.Markdown("The filesystem path of the module where the expression is placed\n\n" +
				modPath),
		},
		{
			Addr: lang.Address{
				lang.RootStep{Name: "path"},
				lang.AttrStep{Name: "root"},
			},
			ScopeId:     builtinScopeId,
			Type:        cty.String,
			Description: lang.Markdown("The filesystem path of the root module of the configuration"),
		},
		{
			Addr: lang.Address{
				lang.RootStep{Name: "path"},
				lang.AttrStep{Name: "cwd"},
			},
			ScopeId: builtinScopeId,
			Type:    cty.String,
			Description: lang.Markdown("The filesystem path of the current working directory.\n\n" +
				"In normal use of Terraform this is the same as `path.root`, " +
				"but some advanced uses of Terraform run it from a directory " +
				"other than the root module directory, causing these paths to be different."),
		},
		{
			Addr: lang.Address{
				lang.RootStep{Name: "terraform"},
				lang.AttrStep{Name: "workspace"},
			},
			ScopeId:     builtinScopeId,
			Type:        cty.String,
			Description: lang.Markdown("The name of the currently selected workspace"),
		},
	}
}
