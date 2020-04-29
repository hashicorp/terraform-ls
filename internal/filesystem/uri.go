package filesystem

import (
	"net/url"
	"path/filepath"
)

func URIFromPath(path string) string {
	p := filepath.ToSlash(path)
	p = wrapPath(p)

	u := &url.URL{
		Scheme: "file",
		Path:   p,
	}
	return u.String()
}
