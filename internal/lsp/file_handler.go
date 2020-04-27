package lsp

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/go-lsp"
)

const uriPrefix = "file://"

type FileHandler string

func (fh FileHandler) Valid() bool {
	if !strings.HasPrefix(string(fh), uriPrefix) {
		return false
	}
	p := string(fh[len(uriPrefix):])
	_, err := url.PathUnescape(p)
	if err != nil {
		return false
	}
	return true
}

func (fh FileHandler) FullPath() string {
	if !fh.Valid() {
		panic("invalid uri")
	}
	p := string(fh[len(uriPrefix):])
	p, _ = url.PathUnescape(p)
	return filepath.FromSlash(p)
}

func (fh FileHandler) Dir() string {
	return filepath.Dir(fh.FullPath())
}

func (fh FileHandler) Filename() string {
	return filepath.Base(fh.FullPath())
}

func (fh FileHandler) DocumentURI() lsp.DocumentURI {
	return lsp.DocumentURI(fh)
}

func (fh FileHandler) URI() string {
	return string(fh)
}

type versionedFileHandler struct {
	FileHandler
	v int
}

func VersionedFileHandler(doc lsp.VersionedTextDocumentIdentifier) *versionedFileHandler {
	return &versionedFileHandler{
		FileHandler: FileHandler(doc.URI),
		v:           doc.Version,
	}
}

func (fh *versionedFileHandler) Version() int {
	return fh.v
}

func FileHandlerFromPath(path string) FileHandler {
	return FileHandler(uriPrefix + filepath.ToSlash(path))
}
