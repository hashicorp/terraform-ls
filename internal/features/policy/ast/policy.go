// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package ast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

type PolicyFilename string

func (mf PolicyFilename) String() string {
	return string(mf)
}

func (mf PolicyFilename) IsJSON() bool {
	return strings.HasSuffix(string(mf), ".json")
}

func (mf PolicyFilename) IsIgnored() bool {
	return globalAst.IsIgnoredFile(string(mf))
}

func IsPolicyFilename(name string) bool {
	return strings.HasSuffix(name, ".policy.hcl") ||
		strings.HasSuffix(name, ".policy.json")
}

type PolicyFiles map[PolicyFilename]*hcl.File

func PolicyFilesFromMap(m map[string]*hcl.File) PolicyFiles {
	mf := make(PolicyFiles, len(m))
	for name, file := range m {
		mf[PolicyFilename(name)] = file
	}
	return mf
}

func (mf PolicyFiles) AsMap() map[string]*hcl.File {
	m := make(map[string]*hcl.File, len(mf))
	for name, file := range mf {
		m[string(name)] = file
	}
	return m
}

func (mf PolicyFiles) Copy() PolicyFiles {
	m := make(PolicyFiles, len(mf))
	for name, file := range mf {
		m[name] = file
	}
	return m
}

type PolicyDiags map[PolicyFilename]hcl.Diagnostics

func PolicyDiagsFromMap(m map[string]hcl.Diagnostics) PolicyDiags {
	mf := make(PolicyDiags, len(m))
	for name, file := range m {
		mf[PolicyFilename(name)] = file
	}
	return mf
}

// AutoloadedOnly returns only diagnostics that are not from ignored files
func (md PolicyDiags) AutoloadedOnly() PolicyDiags {
	diags := make(PolicyDiags)
	for name, f := range md {
		if !name.IsIgnored() {
			diags[name] = f
		}
	}
	return diags
}

func (md PolicyDiags) AsMap() map[string]hcl.Diagnostics {
	m := make(map[string]hcl.Diagnostics, len(md))
	for name, diags := range md {
		m[string(name)] = diags
	}
	return m
}

func (md PolicyDiags) Copy() PolicyDiags {
	m := make(PolicyDiags, len(md))
	for name, diags := range md {
		m[name] = diags
	}
	return m
}

func (md PolicyDiags) Count() int {
	count := 0
	for _, diags := range md {
		count += len(diags)
	}
	return count
}

type SourcePolicyDiags map[globalAst.DiagnosticSource]PolicyDiags

func (smd SourcePolicyDiags) Count() int {
	count := 0
	for _, diags := range smd {
		count += diags.Count()
	}
	return count
}
