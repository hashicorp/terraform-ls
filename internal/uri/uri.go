package uri

import (
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
