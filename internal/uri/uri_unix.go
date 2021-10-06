//go:build !windows
// +build !windows

package uri

import (
	"path/filepath"
)

// wrapPath is no-op for unix-style paths
func wrapPath(path string) string {
	return path
}

func PathFromURI(uri string) (string, error) {
	p, err := parseUri(uri)
	if err != nil {
		return "", err
	}

	return filepath.FromSlash(p), nil
}

func MustPathFromURI(uri string) string {
	p := mustParseUri(uri)
	return filepath.FromSlash(p)
}
