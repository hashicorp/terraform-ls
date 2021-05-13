package uri

import (
	"testing"
)

func TestFromPath(t *testing.T) {
	path := `C:\Users\With Space\file.tf`
	uri := FromPath(path)

	expectedURI := "file:///C:/Users/With%20Space/file.tf"
	if uri != expectedURI {
		t.Fatalf("URI doesn't match.\nExpected: %q\nGiven: %q",
			expectedURI, uri)
	}
}

func TestPathFromURI_valid_windowsFile(t *testing.T) {
	uri := "file:///C:/Users/With%20Space/tf-test/file.tf"
	if !IsURIValid(uri) {
		t.Fatalf("Expected %q to be valid", uri)
	}

	expectedPath := `C:\Users\With Space\tf-test\file.tf`
	path, err := PathFromURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	if path != expectedPath {
		t.Fatalf("Expected full path: %q, given: %q",
			expectedPath, path)
	}
}

func TestPathFromURI_valid_windowsDir(t *testing.T) {
	uri := "file:///C:/Users/With%20Space/tf-test"
	if !IsURIValid(uri) {
		t.Fatalf("Expected %q to be valid", uri)
	}

	expectedPath := `C:\Users\With Space\tf-test`
	path, err := PathFromURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	if path != expectedPath {
		t.Fatalf("Expected full path: %q, given: %q",
			expectedPath, path)
	}
}
