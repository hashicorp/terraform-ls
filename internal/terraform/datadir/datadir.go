// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datadir

import (
	"io/fs"
	"path/filepath"
	"strings"
)

type DataDir struct {
	ModuleManifestPath string
	PluginLockFilePath string
}

type WatchablePaths struct {
	Dirs            []string
	ModuleManifests []string
	PluginLockFiles []string
}

func WatchableModulePaths(modPath string) *WatchablePaths {
	wp := &WatchablePaths{
		Dirs:            watchableModuleDirs(modPath),
		ModuleManifests: make([]string, 0),
		PluginLockFiles: make([]string, 0),
	}

	manifestPath := filepath.Join(append([]string{modPath}, manifestPathElements...)...)
	wp.ModuleManifests = append(wp.ModuleManifests, manifestPath)

	for _, pathElems := range pluginLockFilePathElements {
		filePath := filepath.Join(append([]string{modPath}, pathElems...)...)
		wp.PluginLockFiles = append(wp.PluginLockFiles, filePath)
	}

	return wp
}

// ModulePath strips known lock file paths to get the path
// to the (closest) module these files belong to
func ModulePath(filePath string) (string, bool) {
	manifestSuffix := filepath.Join(manifestPathElements...)
	if strings.HasSuffix(filePath, manifestSuffix) {
		return strings.TrimSuffix(filePath, manifestSuffix), true
	}

	for _, pathElems := range pluginLockFilePathElements {
		suffix := filepath.Join(pathElems...)
		if strings.HasSuffix(filePath, suffix) {
			return strings.TrimSuffix(filePath, suffix), true
		}
	}

	return "", false
}

func WalkDataDirOfModule(fs fs.StatFS, modPath string) *DataDir {
	dir := &DataDir{}

	path, ok := ModuleManifestFilePath(fs, modPath)
	if ok {
		dir.ModuleManifestPath = path
	}

	path, ok = PluginLockFilePath(fs, modPath)
	if ok {
		dir.PluginLockFilePath = path
	}

	return dir
}
