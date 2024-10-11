// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	tfmod "github.com/hashicorp/terraform-schema/module"
)

// InstalledModules is a map of normalized source addresses from the
// manifest to the path of the local directory where the module is installed
type InstalledModules map[string]string

func InstalledModulesFromManifest(manifest *datadir.ModuleManifest) InstalledModules {
	if manifest == nil {
		return nil
	}

	installedModules := make(InstalledModules, len(manifest.Records))

	// TODO: To support multiple versions of the same module, we need to
	// look into resolving the version constraints to a specific version.
	for _, v := range manifest.Records {
		installedModules[v.RawSourceAddr] = v.Dir
	}

	return installedModules
}

func InstalledModulesFromTerraformSources(path string, sources *datadir.TerraformSources) InstalledModules {
	if sources == nil {
		return nil
	}

	// map raw source address to local directory

	installedModules := make(InstalledModules)

	for _, remote := range sources.RemotePackages() {
		absDir, err := sources.LocalPathForSource(remote.SourceAddr(""))
		if err != nil {
			continue // TODO: log error
		}
		// installed modules expects a relative dir
		dir, err := filepath.Rel(path, absDir)
		if err != nil {
			continue // TODO: log error
		}

		normalizedSource := tfmod.ParseModuleSourceAddr(remote.String())
		installedModules[normalizedSource.String()] = dir
	}

	return installedModules
}
