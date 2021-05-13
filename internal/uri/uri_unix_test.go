// +build !windows

package uri

import (
	"testing"
)

func TestURIFromPath(t *testing.T) {
	path := "/random/path"
	uri := FromPath(path)

	expectedURI := "file:///random/path"
	if uri != expectedURI {
		t.Fatalf("URI doesn't match.\nExpected: %q\nGiven: %q",
			expectedURI, uri)
	}
}

func TestPathFromURI_valid_unixFile(t *testing.T) {
	uri := "file:///valid/path/to/file.tf"
	if !IsURIValid(uri) {
		t.Fatalf("Expected %q to be valid", uri)
	}

	expectedFullPath := "/valid/path/to/file.tf"
	path, err := PathFromURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	if path != expectedFullPath {
		t.Fatalf("Expected full path: %q, given: %q",
			expectedFullPath, path)
	}
}

func TestPathFromURI_valid_unixDir(t *testing.T) {
	uri := "file:///valid/path/to"
	expectedDir := "/valid/path/to"
	path, err := PathFromURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	if path != expectedDir {
		t.Fatalf("Expected dir: %q, given: %q",
			expectedDir, path)
	}
}
