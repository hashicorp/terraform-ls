// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

type StackFilename string
type StackFiles map[StackFilename]*hcl.File
type StackDiags map[StackFilename]hcl.Diagnostics

func (mf StackFilename) String() string {
	return string(mf)
}

func (mf StackFilename) IsJSON() bool {
	return strings.HasSuffix(string(mf), ".json")
}

func (mf StackFilename) IsIgnored() bool {
	return globalAst.IsIgnoredFile(string(mf))
}

func IsStackFilename(name string) bool {
	return strings.HasSuffix(name, ".tfstack.hcl") ||
		strings.HasSuffix(name, ".tfstack.json")
}

func (sf StackFiles) Copy() StackFiles {
	m := make(StackFiles, len(sf))
	for name, file := range sf {
		m[name] = file
	}
	return m
}

func (sd StackDiags) Copy() StackDiags {
	m := make(StackDiags, len(sd))
	for name, diags := range sd {
		m[name] = diags
	}
	return m
}

// AutoloadedOnly returns only diagnostics that are not from ignored files
func (sd StackDiags) AutoloadedOnly() StackDiags {
	diags := make(StackDiags)
	for name, f := range sd {
		if !name.IsIgnored() {
			diags[name] = f
		}
	}
	return diags
}

func (sd StackDiags) AsMap() map[string]hcl.Diagnostics {
	m := make(map[string]hcl.Diagnostics, len(sd))
	for name, diags := range sd {
		m[string(name)] = diags
	}
	return m
}

func (sd StackDiags) Count() int {
	count := 0
	for _, diags := range sd {
		count += len(diags)
	}
	return count
}

type SourceStackDiags map[globalAst.DiagnosticSource]StackDiags

func (svd SourceStackDiags) Count() int {
	count := 0
	for _, diags := range svd {
		count += diags.Count()
	}
	return count
}
