package lsp

import (
	"net/url"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/sourcegraph/go-lsp"
)

type FileHandler string

func (fh FileHandler) Valid() bool {
	_, err := fh.parsePath()
	if err != nil {
		return false
	}

	return true
}

func (fh FileHandler) parsePath() (string, error) {
	u, err := url.ParseRequestURI(string(fh))
	if err != nil {
		return "", err
	}

	return url.PathUnescape(u.Path)
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
	return FileHandler(filesystem.URIFromPath(path))
}
