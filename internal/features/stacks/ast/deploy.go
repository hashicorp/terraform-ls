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

func (mf DeployFilename) String() string {
	return string(mf)
}

func (mf DeployFilename) IsJSON() bool {
	return strings.HasSuffix(string(mf), ".json")
}

func (mf DeployFilename) IsIgnored() bool {
	return globalAst.IsIgnoredFile(string(mf))
}

func IsDeployFilename(name string) bool {
	return strings.HasSuffix(name, ".tfdeploy.hcl") ||
		strings.HasSuffix(name, ".tfdeploy.json")
}

func (sf DeployFiles) Copy() DeployFiles {
	m := make(DeployFiles, len(sf))
	for name, file := range sf {
		m[name] = file
	}
	return m
}

func (sd DeployDiags) Copy() DeployDiags {
	m := make(DeployDiags, len(sd))
	for name, diags := range sd {
		m[name] = diags
	}
	return m
}
