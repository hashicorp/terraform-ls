package filesystem

import (
	"net/url"
	"path/filepath"
	"strings"

	lsp "github.com/sourcegraph/go-lsp"
)

const uriPrefix = "file://"

type URI string

func (u URI) Valid() bool {
	if !strings.HasPrefix(string(u), uriPrefix) {
		return false
	}
	p := string(u[len(uriPrefix):])
	_, err := url.PathUnescape(p)
	if err != nil {
		return false
	}
	return true
}

func (u URI) FullPath() string {
	if !u.Valid() {
		panic("invalid uri")
	}
	p := string(u[len(uriPrefix):])
	p, _ = url.PathUnescape(p)
	return filepath.FromSlash(p)
}

func (u URI) Dir() string {
	return filepath.Dir(u.FullPath())
}

func (u URI) Filename() string {
	return filepath.Base(u.FullPath())
}

func (u URI) PathParts() (full, dir, filename string) {
	full = u.FullPath()
	dir = filepath.Dir(full)
	filename = filepath.Base(full)
	return full, dir, filename
}

func (u URI) LSPDocumentURI() lsp.DocumentURI {
	return lsp.DocumentURI(u)
}
