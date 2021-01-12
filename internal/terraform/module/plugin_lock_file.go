package module

import (
	"path/filepath"
	"runtime"
)

func pluginLockFilePaths(dir string) []string {
	return []string{
		// Terraform >= 0.14
		filepath.Join(dir,
			".terraform.lock.hcl",
		),
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
