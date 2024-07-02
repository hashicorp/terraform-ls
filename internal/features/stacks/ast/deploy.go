// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

type DeployFilename string
type DeployFiles map[DeployFilename]*hcl.File
type DeployDiags map[DeployFilename]hcl.Diagnostics
type SourceDeployDiags map[globalAst.DiagnosticSource]DeployDiags

func (df DeployFilename) String() string {
	return string(df)
}

func (df DeployFilename) IsJSON() bool {
	return strings.HasSuffix(string(df), ".json")
}

func (df DeployFilename) IsIgnored() bool {
	return globalAst.IsIgnoredFile(string(df))
}

func IsDeployFilename(name string) bool {
	return strings.HasSuffix(name, ".tfdeploy.hcl") ||
		strings.HasSuffix(name, ".tfdeploy.json")
}

func (df DeployFiles) Copy() DeployFiles {
	m := make(DeployFiles, len(df))
	for name, file := range df {
		m[name] = file
	}
	return m
}

func (dd DeployDiags) Copy() DeployDiags {
	m := make(DeployDiags, len(dd))
	for name, diags := range dd {
		m[name] = diags
	}
	return m
}

func (dd DeployDiags) AutoloadedOnly() DeployDiags {
	diags := make(DeployDiags)
	for name, f := range dd {
		if !name.IsIgnored() {
			diags[name] = f
		}
	}
	return diags
}

func (dd DeployDiags) AsMap() map[string]hcl.Diagnostics {
	m := make(map[string]hcl.Diagnostics, len(dd))
	for name, diags := range dd {
		m[string(name)] = diags
	}
	return m
}

func (dd DeployDiags) Count() int {
	count := 0
	for _, diags := range dd {
		count += len(diags)
	}
	return count
}
