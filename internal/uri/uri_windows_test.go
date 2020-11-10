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
