package datadir

import (
	"path/filepath"
	"runtime"
)

const DataDirName = ".terraform"

var pluginLockFilePathElements = [][]string{
	// Terraform >= 0.14
	{".terraform.lock.hcl"},
	// Terraform >= v0.13
	{DataDirName, "plugins", "selections.json"},
	// Terraform >= v0.12
	{DataDirName, "plugins", runtime.GOOS + "_" + runtime.GOARCH, "lock.json"},
}

var manifestPathElements = []string{
	DataDirName, "modules", "modules.json",
}

func watchableModuleDirs(modPath string) []string {
	return []string{
		filepath.Join(modPath, DataDirName),
		filepath.Join(modPath, DataDirName, "modules"),
		filepath.Join(modPath, DataDirName, "plugins"),
		filepath.Join(modPath, DataDirName, "plugins", runtime.GOOS+"_"+runtime.GOARCH),
	}
}
