package ast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type ModFilename string

func (mf ModFilename) String() string {
	return string(mf)
}

func IsModuleFilename(name string) bool {
	return strings.HasSuffix(name, ".tf") && !isIgnoredFile(name)
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

type ModDiags map[ModFilename]hcl.Diagnostics

func ModDiagsFromMap(m map[string]hcl.Diagnostics) ModDiags {
	mf := make(ModDiags, len(m))
	for name, file := range m {
		mf[ModFilename(name)] = file
	}
	return mf
}

func (md ModDiags) AsMap() map[string]hcl.Diagnostics {
	m := make(map[string]hcl.Diagnostics, len(md))
	for name, diags := range md {
		m[string(name)] = diags
	}
	return m
}
