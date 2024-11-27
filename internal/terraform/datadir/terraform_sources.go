// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datadir

import (
	"path/filepath"

	"github.com/hashicorp/go-slug/sourcebundle"
)

var terraformSourcesDirElements = []string{
	DataDirName, "modules",
}
var terraformSourcesPathElements = []string{
	DataDirName, "modules", "terraform-sources.json",
}

type TerraformSources struct {
	sourcebundle.Bundle
	rootDir string // we need to duplicate this as our rootDir is different from the bundle's
}

func (mm *TerraformSources) Copy() *TerraformSources {
	if mm == nil {
		return nil
	}

	newTfS := &TerraformSources{
		Bundle:  mm.Bundle,
		rootDir: mm.rootDir,
	}

	return newTfS
}

func (tfs *TerraformSources) RootDir() string {
	return tfs.rootDir
}

func ParseTerraformSourcesFromFile(modulePath string) (*TerraformSources, error) {
	path := filepath.Join(
		append([]string{modulePath},
			terraformSourcesDirElements...)...)

	bundle, err := sourcebundle.OpenDir(path)
	if err != nil {
		return nil, err
	}

	tfs := &TerraformSources{
		Bundle: *bundle,
	}

	tfs.rootDir = filepath.Clean(modulePath)

	return tfs, nil
}
