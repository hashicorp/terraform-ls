package lsp

import (
	"path/filepath"
	"strings"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
)

func FileHandlerFromDocumentURI(docUri lsp.DocumentURI) *fileHandler {
	return &fileHandler{uri: string(docUri)}
}

func FileHandlerFromDirURI(dirUri lsp.DocumentURI) *fileHandler {
	// Dir URIs are usually without trailing separator already
	// but we do sanity check anyway, so we deal with the same URI
	// regardless of language client differences
	uri := strings.TrimSuffix(string(dirUri), "/")
	return &fileHandler{uri: uri, isDir: true}
}

type FileHandler interface {
	Valid() bool
	Dir() string
	IsDir() bool
	Filename() string
	DocumentURI() lsp.DocumentURI
	URI() string
}

type fileHandler struct {
	uri   string
	isDir bool
}

func (fh *fileHandler) Valid() bool {
	return uri.IsURIValid(fh.uri)
}

func (fh *fileHandler) IsDir() bool {
	return fh.isDir
}

func (fh *fileHandler) Dir() string {
	if fh.isDir {
		return fh.FullPath()
	}

	path := filepath.Dir(fh.FullPath())
	return path
}

func (fh *fileHandler) Filename() string {
	return filepath.Base(fh.FullPath())
}

func (fh *fileHandler) FullPath() string {
	return uri.MustPathFromURI(fh.uri)
}

func (fh *fileHandler) DocumentURI() lsp.DocumentURI {
	return lsp.DocumentURI(fh.uri)
}

func (fh *fileHandler) URI() string {
	return string(fh.uri)
}

func (fh *fileHandler) LanguageID() string {
	return ""
}

type versionedFileHandler struct {
	fileHandler
	v int
}

func VersionedFileHandler(doc lsp.VersionedTextDocumentIdentifier) *versionedFileHandler {
	return &versionedFileHandler{
		fileHandler: fileHandler{uri: string(doc.URI)},
		v:           int(doc.Version),
	}
}

func (fh *versionedFileHandler) Version() int {
	return fh.v
}

func FileHandlerFromPath(path string) *fileHandler {
	return &fileHandler{uri: uri.FromPath(path)}
}

func FileHandlerFromDirPath(dirPath string) *fileHandler {
	// Dir URIs are usually without trailing separator in LSP
	dirPath = filepath.Clean(dirPath)

	return &fileHandler{uri: uri.FromPath(dirPath), isDir: true}
}
