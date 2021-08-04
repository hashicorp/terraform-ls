package uri

import (
	"fmt"
	"net/url"
	"path/filepath"
)

func FromPath(path string) string {
	p := filepath.ToSlash(path)
	p = wrapPath(p)

	u := &url.URL{
		Scheme: "file",
		Path:   p,
	}
	return u.String()
}

func IsURIValid(uri string) bool {
	_, err := parseUri(uri)
	if err != nil {
		return false
	}

	return true
}

func mustParseUri(uri string) string {
	u, err := parseUri(uri)
	if err != nil {
		panic(fmt.Sprintf("invalid URI: %s", uri))
	}
	return u
}

func parseUri(uri string) (string, error) {
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return "", err
	}

	if u.Scheme != "file" {
		return "", fmt.Errorf("unexpected scheme %q in URI %q",
			u.Scheme, uri)
	}

	return url.PathUnescape(u.Path)
}
