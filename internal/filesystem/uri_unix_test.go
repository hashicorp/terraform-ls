// +build !windows

package filesystem

import (
	"testing"
)

func TestURIFromPath(t *testing.T) {
	path := "/random/path"
	uri := URIFromPath(path)

	expectedURI := "file:///random/path"
	if uri != expectedURI {
		t.Fatalf("URI doesn't match.\nExpected: %q\nGiven: %q",
			expectedURI, uri)
	}
}
