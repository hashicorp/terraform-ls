package addrs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	// "github.com/hashicorp/hcl/v2"
)

func TestParseProviderConfigCompact_empty(t *testing.T) {
	lProviderCfg, err := ParseProviderConfigCompact(nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := LocalProviderConfig{}
	if diff := cmp.Diff(expected, lProviderCfg); diff != "" {
		t.Fatalf("mismatch: %s", diff)
	}
}

func TestParseProviderConfigCompactStr_empty(t *testing.T) {
	_, err := ParseProviderConfigCompactStr("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestParseProviderConfigCompactStr_nameOnly(t *testing.T) {
	lProviderCfg, err := ParseProviderConfigCompactStr("justname")
	if err != nil {
		t.Fatal(err)
	}
	expected := LocalProviderConfig{LocalName: "justname"}
	if diff := cmp.Diff(expected, lProviderCfg); diff != "" {
		t.Fatalf("mismatch: %s", diff)
	}
}

func TestParseProviderConfigCompactStr_fullRef(t *testing.T) {
	lProviderCfg, err := ParseProviderConfigCompactStr("aws.uswest")
	if err != nil {
		t.Fatal(err)
	}
	expected := LocalProviderConfig{LocalName: "aws", Alias: "uswest"}
	if diff := cmp.Diff(expected, lProviderCfg); diff != "" {
		t.Fatalf("mismatch: %s", diff)
	}
}
