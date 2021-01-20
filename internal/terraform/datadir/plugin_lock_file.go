package datadir

import (
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
)

func PluginLockFilePath(fs filesystem.Filesystem, modPath string) (string, bool) {
	for _, pathElems := range pluginLockFilePathElements {
		fullPath := filepath.Join(append([]string{modPath}, pathElems...)...)
		fi, err := fs.Stat(fullPath)
		if err == nil && fi.Mode().IsRegular() {
			return fullPath, true
		}
	}

	return "", false
}
