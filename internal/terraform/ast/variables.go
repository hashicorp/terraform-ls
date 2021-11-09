package ast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type VarsFilename string

func NewVarsFilename(name string) (VarsFilename, bool) {
	if IsVarsFilename(name) {
		return VarsFilename(name), true
	}
	return "", false
}

func IsVarsFilename(name string) bool {
	return (strings.HasSuffix(name, ".tfvars") ||
		strings.HasSuffix(name, ".tfvars.json")) &&
		!isIgnoredFile(name)
}

func (vf VarsFilename) String() string {
	return string(vf)
}

func (vf VarsFilename) IsJSON() bool {
	return strings.HasSuffix(string(vf), ".json")
}

func (vf VarsFilename) IsAutoloaded() bool {
	name := string(vf)
	return strings.HasSuffix(name, ".auto.tfvars") ||
		strings.HasSuffix(name, ".auto.tfvars.json") ||
		name == "terraform.tfvars" ||
		name == "terraform.tfvars.json"
}

type VarsFiles map[VarsFilename]*hcl.File

func VarsFilesFromMap(m map[string]*hcl.File) VarsFiles {
	mf := make(VarsFiles, len(m))
	for name, file := range m {
		mf[VarsFilename(name)] = file
	}
	return mf
}

type VarsDiags map[VarsFilename]hcl.Diagnostics

func VarsDiagsFromMap(m map[string]hcl.Diagnostics) VarsDiags {
	mf := make(VarsDiags, len(m))
	for name, file := range m {
		mf[VarsFilename(name)] = file
	}
	return mf
}

func (vd VarsDiags) AutoloadedOnly() VarsDiags {
	diags := make(VarsDiags)
	for name, f := range vd {
		if name.IsAutoloaded() {
			diags[name] = f
		}
	}
	return diags
}

func (vd VarsDiags) ForFile(name VarsFilename) VarsDiags {
	diags := make(VarsDiags)
	for fName, f := range vd {
		if fName == name {
			diags[fName] = f
		}
	}
	return diags
}

func (vd VarsDiags) AsMap() map[string]hcl.Diagnostics {
	m := make(map[string]hcl.Diagnostics, len(vd))
	for name, diags := range vd {
		m[string(name)] = diags
	}
	return m
}
