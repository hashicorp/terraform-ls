// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

type ModFilename string

func (mf ModFilename) String() string {
	return string(mf)
}

func (mf ModFilename) IsJSON() bool {
	return strings.HasSuffix(string(mf), ".json")
}

func (mf ModFilename) IsIgnored() bool {
	return globalAst.IsIgnoredFile(string(mf))
}

func IsModuleFilename(name string) bool {
	return strings.HasSuffix(name, ".tf") ||
		strings.HasSuffix(name, ".tf.json")
}

type ModFiles map[ModFilename]*hcl.File

func ModFilesFromMap(m map[string]*hcl.File) ModFiles {
	mf := make(ModFiles, len(m))
	for name, file := range m {
		mf[ModFilename(name)] = file
	}
	return mf
}

func (mf ModFiles) AsMap() map[string]*hcl.File {
	m := make(map[string]*hcl.File, len(mf))
	for name, file := range mf {
		m[string(name)] = file
	}
	return m
}

func (mf ModFiles) Copy() ModFiles {
	m := make(ModFiles, len(mf))
	for name, file := range mf {
		m[name] = file
	}
	return m
}

type ModDiags map[ModFilename]hcl.Diagnostics

func ModDiagsFromMap(m map[string]hcl.Diagnostics) ModDiags {
	mf := make(ModDiags, len(m))
	for name, file := range m {
		mf[ModFilename(name)] = file
	}
	return mf
}

// AutoloadedOnly returns only diagnostics that are not from ignored files
func (md ModDiags) AutoloadedOnly() ModDiags {
	diags := make(ModDiags)
	for name, f := range md {
		if !name.IsIgnored() {
			diags[name] = f
		}
	}
	return diags
}

func (md ModDiags) AsMap() map[string]hcl.Diagnostics {
	m := make(map[string]hcl.Diagnostics, len(md))
	for name, diags := range md {
		m[string(name)] = diags
	}
	return m
}

func (md ModDiags) Copy() ModDiags {
	m := make(ModDiags, len(md))
	for name, diags := range md {
		m[name] = diags
	}
	return m
}

func (md ModDiags) Count() int {
	count := 0
	for _, diags := range md {
		count += len(diags)
	}
	return count
}

type SourceModDiags map[globalAst.DiagnosticSource]ModDiags

func (smd SourceModDiags) Count() int {
	count := 0
	for _, diags := range smd {
		count += diags.Count()
	}
	return count
}
