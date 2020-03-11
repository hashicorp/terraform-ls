package lang

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestParser_ParseBlockFromHCL(t *testing.T) {
	p := newParser()

	hclBlock := parseHclBlock(t, `provider "currywurst" {
}`)

	cfgBlock, err := p.ParseBlockFromHCL(hclBlock)
	if err != nil {
		t.Fatal(err)
	}

	blockType := cfgBlock.BlockType()
	if blockType != "provider" {
		t.Fatalf("Expected block type to be 'provider', given: %q",
			blockType)
	}
}

func TestParser_ParseBlockFromHCL_unknown(t *testing.T) {
	p := newParser()

	hclBlock := parseHclBlock(t, `meal "currywurst" {
}`)

	_, err := p.ParseBlockFromHCL(hclBlock)

	expectedErr := `unknown block type: "meal"`
	if err != nil {
		if err.Error() != expectedErr {
			t.Fatalf("error doesn't match.\nexpected: %q\ngiven: %q\n",
				expectedErr, err.Error())
		}
		return
	}
	t.Fatalf("expected error: %q", expectedErr)
}

func TestParser_ParseBlockFromHCL_error(t *testing.T) {
	p := newParser()

	hclBlock := parseHclBlock(t, `provider "currywurst" "extra" {
}`)

	_, err := p.ParseBlockFromHCL(hclBlock)

	expectedErr := `provider: invalid labels for provider block: ["currywurst" "extra"]`
	if err != nil {
		if err.Error() != expectedErr {
			t.Fatalf("error doesn't match.\nexpected: %q\ngiven: %q\n",
				expectedErr, err.Error())
		}
		return
	}
	t.Fatalf("expected error: %q", expectedErr)
}

func parseHclBlock(t *testing.T, src string) *hcl.Block {
	f, diags := hclsyntax.ParseConfig([]byte(src), "/test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	return f.OutermostBlockAtPos(hcl.InitialPos)
}
