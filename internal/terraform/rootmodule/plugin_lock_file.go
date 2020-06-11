package rootmodule

import (
	"path/filepath"
	"runtime"
)

func pluginLockFilePaths(dir string) []string {
	return []string{
		// Terraform >= v0.13
		filepath.Join(dir,
			".terraform",
			"plugins",
			"selections.json"),
		// Terraform <= v0.12
		filepath.Join(dir,
			".terraform",
			"plugins",
			runtime.GOOS+"_"+runtime.GOARCH,
			"lock.json"),
	}
}
