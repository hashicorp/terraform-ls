package filesystem

import (
	"testing"
)

func TestURIFromPath(t *testing.T) {
	path := `C:\Users\With Space\file.tf`
	uri := URIFromPath(path)

	expectedURI := "file:///C:/Users/With%20Space/file.tf"
	if uri != expectedURI {
		t.Fatalf("URI doesn't match.\nExpected: %q\nGiven: %q",
			expectedURI, uri)
	}
}
