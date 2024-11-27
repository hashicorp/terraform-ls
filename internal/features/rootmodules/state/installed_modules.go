// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"log"
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

func InstalledModulesFromTerraformSources(path string, sources *datadir.TerraformSources, logger *log.Logger) InstalledModules {
	if sources == nil {
		return nil
	}

	// map raw source address to local directory

	installedModules := make(InstalledModules)

	for _, remote := range sources.RemotePackages() {
		absDir, err := sources.LocalPathForSource(remote.SourceAddr(""))
		if err != nil {
			logger.Printf("Error getting local path for source %s: %s", remote.String(), err)
			continue
		}
		// installed modules expects a relative dir
		dir, err := filepath.Rel(path, absDir)
		if err != nil {
			logger.Printf("Error getting relative path for source %s and path %s and absolute dir %s: %s", remote.String(), path, absDir, err)
			continue
		}

		normalizedSource := tfmod.ParseModuleSourceAddr(remote.String())
		installedModules[normalizedSource.String()] = dir
	}

	for _, pkg := range sources.RegistryPackages() {
		for _, version := range sources.RegistryPackageVersions(pkg) {
			addr, ok := sources.RegistryPackageSourceAddr(pkg, version)

			if !ok {
				logger.Printf("Error getting source address for package %s and version %s", pkg.String(), version.String())
				continue
			}

			absDir, err := sources.LocalPathForSource(addr)
			if err != nil {
				logger.Printf("Error getting local path for source %s: %s", pkg.String(), err)
				continue
			}
			// installed modules expects a relative dir
			dir, err := filepath.Rel(path, absDir)
			if err != nil {
				logger.Printf("Error getting relative path for source %s and path %s and absolute dir %s: %s", pkg.String(), path, absDir, err)
				continue
			}

			normalizedSource := tfmod.ParseModuleSourceAddr(pkg.String())
			installedModules[normalizedSource.String()] = dir
		}
	}

	return installedModules
}
