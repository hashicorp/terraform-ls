// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package parser

import (
	"io/fs"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/json"
)

type FS interface {
	fs.FS
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

type filename interface {
	IsJSON() bool
	String() string
}

func parseFile(src []byte, filename filename) (*hcl.File, hcl.Diagnostics) {
	if filename.IsJSON() {
		return json.Parse(src, filename.String())
	}
	return hclsyntax.ParseConfig(src, filename.String(), hcl.InitialPos)
}
