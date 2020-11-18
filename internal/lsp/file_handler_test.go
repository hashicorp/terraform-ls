package lsp

import (
	"testing"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

var (
	validUnixPath = "file:///valid/path/to/file.tf"
)

func TestFileHandler_invalid(t *testing.T) {
	path := "invalidpath"
	fh := FileHandlerFromDocumentURI(lsp.DocumentURI(path))
	if fh.Valid() {
		t.Fatalf("Expected %q to be invalid", path)
	}
}
