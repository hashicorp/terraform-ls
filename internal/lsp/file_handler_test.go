package lsp

import (
	"testing"

	"github.com/sourcegraph/go-lsp"
)

var (
	validUnixPath    = "file:///valid/path/to/file.tf"
	validWindowsPath = "file:///C:/Users/With%20Space/tf-test/file.tf"
)

func TestFileHandler_invalid(t *testing.T) {
	path := "invalidpath"
	fh := FileHandlerFromDocumentURI(lsp.DocumentURI(path))
	if fh.Valid() {
		t.Fatalf("Expected %q to be invalid", path)
	}
}
