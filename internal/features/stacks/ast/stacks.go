// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	globalAst "github.com/hashicorp/terraform-ls/internal/terraform/ast"
)

type Filename interface {
	String() string
	IsJSON() bool
	IsIgnored() bool
}

// StackFilename is a custom type for stack configuration files
type StackFilename string

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
		strings.HasSuffix(name, ".tfstack.json") ||
		strings.HasSuffix(name, ".tfcomponent.hcl") ||
		strings.HasSuffix(name, ".tfcomponent.json")
}

// DeployFilename is a custom type for deployment files
type DeployFilename string

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

// FilenameFromName returns either a StackFilename or DeployFilename based
// on the name
func FilenameFromName(name string) Filename {
	if IsStackFilename(name) {
		return StackFilename(name)
	}
	if IsDeployFilename(name) {
		return DeployFilename(name)
	}

	return nil
}

type Files map[Filename]*hcl.File

func (sf Files) Copy() Files {
	m := make(Files, len(sf))
	for name, file := range sf {
		m[name] = file
	}
	return m
}

func (mf Files) AsMap() map[string]*hcl.File {
	m := make(map[string]*hcl.File, len(mf))
	for name, file := range mf {
		m[name.String()] = file // TODO: is this right?
	}
	return m
}

type Diagnostics map[Filename]hcl.Diagnostics

func (sd Diagnostics) Copy() Diagnostics {
	m := make(Diagnostics, len(sd))
	for name, diags := range sd {
		m[name] = diags
	}
	return m
}

// AutoloadedOnly returns only diagnostics that are not from ignored files
func (sd Diagnostics) AutoloadedOnly() Diagnostics {
	diags := make(Diagnostics)
	for name, f := range sd {
		if !name.IsIgnored() {
			diags[name] = f
		}
	}
	return diags
}

func (sd Diagnostics) AsMap() map[string]hcl.Diagnostics {
	m := make(map[string]hcl.Diagnostics, len(sd))
	for name, diags := range sd {
		m[name.String()] = diags
	}
	return m
}

func (sd Diagnostics) Count() int {
	count := 0
	for _, diags := range sd {
		count += len(diags)
	}
	return count
}

func DiagnosticsFromMap(m map[string]hcl.Diagnostics) Diagnostics {
	mf := make(Diagnostics, len(m))
	for name, file := range m {
		mf[FilenameFromName(name)] = file
	}
	return mf
}

type SourceDiagnostics map[globalAst.DiagnosticSource]Diagnostics

func (svd SourceDiagnostics) Count() int {
	count := 0
	for _, diags := range svd {
		count += diags.Count()
	}
	return count
}
