// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import "github.com/hashicorp/terraform-ls/internal/terraform/datadir"

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
