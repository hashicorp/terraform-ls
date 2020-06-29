package lsp

import (
	"path/filepath"
	"strings"
)

// FullPath on Windows strips the leading '/'
// which occurs in Windows-style paths (e.g. file:///C:/)
// as url.URL methods don't account for that
// (see golang/go#6027).
func (fh *fileHandler) FullPath() string {
	p, err := fh.parsePath()
	if err != nil {
		panic("invalid uri")
	}

	p = strings.TrimPrefix(p, "/")

	return filepath.FromSlash(p)
}
