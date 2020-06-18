// +build !windows

package lsp

import (
	"testing"

	"github.com/sourcegraph/go-lsp"
)

func TestFileHandler_valid_unix(t *testing.T) {
	fh := FileHandlerFromDocumentURI(lsp.DocumentURI(validUnixPath))
	if !fh.Valid() {
		t.Fatalf("Expected %q to be valid", validUnixPath)
	}

	expectedDir := "/valid/path/to"
	if fh.Dir() != expectedDir {
		t.Fatalf("Expected dir: %q, given: %q",
			expectedDir, fh.Dir())
	}

	expectedFilename := "file.tf"
	if fh.Filename() != expectedFilename {
		t.Fatalf("Expected filename: %q, given: %q",
			expectedFilename, fh.Filename())
	}

	expectedFullPath := "/valid/path/to/file.tf"
	if fh.FullPath() != expectedFullPath {
		t.Fatalf("Expected full path: %q, given: %q",
			expectedFullPath, fh.FullPath())
	}

	if fh.URI() != validUnixPath {
		t.Fatalf("Expected document URI: %q, given: %q",
			validUnixPath, fh.URI())
	}
}

func TestFileHandler_valid_unixDir(t *testing.T) {
	fh := FileHandlerFromDirURI(lsp.DocumentURI("/valid/path/to"))
	if !fh.Valid() {
		t.Fatalf("Expected %q to be valid", "/valid/path/to")
	}

	expectedDir := "/valid/path/to"
	if fh.Dir() != expectedDir {
		t.Fatalf("Expected dir: %q, given: %q",
			expectedDir, fh.Dir())
	}
}
