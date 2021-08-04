package uri

import (
	"testing"
)

func TestIsURIValid_invalid(t *testing.T) {
	uri := "output:extension-output-%232"
	if IsURIValid(uri) {
		t.Fatalf("Expected %q to be invalid", uri)
	}
}
