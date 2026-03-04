// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package ast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

type PolicyTestFilename string

func (mf PolicyTestFilename) String() string {
	return string(mf)
}

func (mf PolicyTestFilename) IsJSON() bool {
	return strings.HasSuffix(string(mf), ".json")
}

func (mf PolicyTestFilename) IsIgnored() bool {
	return globalAst.IsIgnoredFile(string(mf))
}

func IsPolicyTestFilename(name string) bool {
	return strings.HasSuffix(name, ".policytest.hcl") ||
		strings.HasSuffix(name, ".policytest.json")
}

type PolicyTestFiles map[PolicyTestFilename]*hcl.File

func PolicyTestFilesFromMap(m map[string]*hcl.File) PolicyTestFiles {
	mf := make(PolicyTestFiles, len(m))
	for name, file := range m {
		mf[PolicyTestFilename(name)] = file
	}
	return mf
}

func (mf PolicyTestFiles) AsMap() map[string]*hcl.File {
	m := make(map[string]*hcl.File, len(mf))
	for name, file := range mf {
		m[string(name)] = file
	}
	return m
}

func (mf PolicyTestFiles) Copy() PolicyTestFiles {
	m := make(PolicyTestFiles, len(mf))
	for name, file := range mf {
		m[name] = file
	}
	return m
}

type PolicyTestDiags map[PolicyTestFilename]hcl.Diagnostics

func PolicyTestDiagsFromMap(m map[string]hcl.Diagnostics) PolicyTestDiags {
	mf := make(PolicyTestDiags, len(m))
	for name, file := range m {
		mf[PolicyTestFilename(name)] = file
	}
	return mf
}

// AutoloadedOnly returns only diagnostics that are not from ignored files
func (md PolicyTestDiags) AutoloadedOnly() PolicyTestDiags {
	diags := make(PolicyTestDiags)
	for name, f := range md {
		if !name.IsIgnored() {
			diags[name] = f
		}
	}
	return diags
}

func (md PolicyTestDiags) AsMap() map[string]hcl.Diagnostics {
	m := make(map[string]hcl.Diagnostics, len(md))
	for name, diags := range md {
		m[string(name)] = diags
	}
	return m
}

func (md PolicyTestDiags) Copy() PolicyTestDiags {
	m := make(PolicyTestDiags, len(md))
	for name, diags := range md {
		m[name] = diags
	}
	return m
}

func (md PolicyTestDiags) Count() int {
	count := 0
	for _, diags := range md {
		count += len(diags)
	}
	return count
}

type SourcePolicyTestDiags map[globalAst.DiagnosticSource]PolicyTestDiags

func (smd SourcePolicyTestDiags) Count() int {
	count := 0
	for _, diags := range smd {
		count += diags.Count()
	}
	return count
}
