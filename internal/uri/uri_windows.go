package uri

import (
	"path/filepath"
	"strings"
)

// wrapPath prepends Windows-style paths (C:\path)
// with an additional slash to account for an empty hostname
// in a valid file-scheme URI per RFC 8089
func wrapPath(path string) string {
	return "/" + path
}

func PathFromURI(uri string) (string, error) {
	p, err := parseUri(uri)
	if err != nil {
		return "", err
	}

	p = strings.TrimPrefix(p, "/")

	return filepath.FromSlash(p), nil
}

// MustPathFromURI on Windows strips the leading '/'
// which occurs in Windows-style paths (e.g. file:///C:/)
// as url.URL methods don't account for that
// (see golang/go#6027).
func MustPathFromURI(uri string) string {
	p := mustParseUri(uri)

	p = strings.TrimPrefix(p, "/")

	return filepath.FromSlash(p)
}
