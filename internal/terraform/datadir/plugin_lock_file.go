package datadir

import (
	"io/fs"
	"path/filepath"
)

func PluginLockFilePath(fs fs.StatFS, modPath string) (string, bool) {
	for _, pathElems := range pluginLockFilePathElements {
		fullPath := filepath.Join(append([]string{modPath}, pathElems...)...)
		fi, err := fs.Stat(fullPath)
		if err == nil && fi.Mode().IsRegular() {
			return fullPath, true
		}
	}

	return "", false
}
