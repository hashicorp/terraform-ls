// +build !windows

package lsp

import (
	"path/filepath"
)

func (fh *fileHandler) FullPath() string {
	p, err := fh.parsePath()
	if err != nil {
		panic("invalid uri")
	}

	return filepath.FromSlash(p)
}
