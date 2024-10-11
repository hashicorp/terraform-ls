// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datadir

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/hashicorp/go-slug/sourcebundle"
)

var terraformSourcesDirElements = []string{
	DataDirName, "modules",
}
var terraformSourcesPathElements = []string{
	DataDirName, "modules", "terraform-sources.json",
}

func TerraformSourcesDirPath(fs fs.StatFS, modulePath string) (string, bool) {
	terraformSourcesPath := filepath.Join(
		append([]string{modulePath},
			terraformSourcesPathElements...)...)
	terraformSourcesDirPath := filepath.Join(
		append([]string{modulePath},
			terraformSourcesDirElements...)...)

	fi, err := fs.Stat(terraformSourcesPath)
	if err == nil && fi.Mode().IsRegular() {
		return terraformSourcesDirPath, true // TODO: this is a bit weird and misleading, maybe we should just use the bundle thing reading the dir and catch the proper error or sth like that
	}
	return "", false
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

func ParseTerraformSourcesFromFile(path string) (*TerraformSources, error) {
	bundle, err := sourcebundle.OpenDir(path)
	if err != nil {
		return nil, err
	}

	tfs := &TerraformSources{
		Bundle: *bundle,
	}

	rootDir, ok := ModulePath(path)
	if !ok {
		return nil, fmt.Errorf("failed to detect module path: %s", path)
	}
	tfs.rootDir = filepath.Clean(rootDir)

	return tfs, nil
}
