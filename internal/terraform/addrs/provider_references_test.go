package addrs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProviderReferences_LocalNameByAddr(t *testing.T) {
	ref := LocalProviderConfig{LocalName: "customname"}
	addr := Provider{
		Type:      "aws",
		Hostname:  "registry.terraform.io",
		Namespace: "hashicorp",
	}
	refs := ProviderReferences{ref: addr}

	foundRef, ok := refs.LocalNameByAddr(addr)
	if !ok {
		t.Fatal("expected to find the reference")
	}
	if diff := cmp.Diff(ref, foundRef); diff != "" {
		t.Fatalf("reference mismatch: %s", diff)
	}
}
