package lsp

import (
	"testing"
)

func TestFileHandler_invalid(t *testing.T) {
	path := "invalidpath"
	fh := FileHandler(path)
	if fh.Valid() {
		t.Fatalf("Expected %q to be invalid", path)
	}
}

func TestFileHandler_valid(t *testing.T) {
	path := "file://valid/path/to/file.tf"
	fh := FileHandler(path)
	if !fh.Valid() {
		t.Fatalf("Expected %q to be valid", path)
	}

	expectedDir := "valid/path/to"
	if fh.Dir() != expectedDir {
		t.Fatalf("Expected dir: %q, given: %q",
			expectedDir, fh.Dir())
	}

	expectedFilename := "file.tf"
	if fh.Filename() != expectedFilename {
		t.Fatalf("Expected filename: %q, given: %q",
			expectedFilename, fh.Filename())
	}

	expectedFullPath := "valid/path/to/file.tf"
	if fh.FullPath() != expectedFullPath {
		t.Fatalf("Expected full path: %q, given: %q",
			expectedFullPath, fh.FullPath())
	}

	expectedDocumentURI := "file://valid/path/to/file.tf"
	if fh.DocumentURI() != expectedDocumentURI {
		t.Fatalf("Expected document URI: %q, given: %q",
			expectedDocumentURI, fh.DocumentURI())
	}
}
