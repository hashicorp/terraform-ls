package lang

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sourcegraph/go-lsp"
)

func TestParser_ParseBlockFromHcl(t *testing.T) {
	caps := lsp.TextDocumentClientCapabilities{}
	p := NewParser(emptyLogger(), caps)

	hclBlock := parseHclBlock(t, `provider "currywurst" {
}`)

	cfgBlock, err := p.ParseBlockFromHcl(hclBlock)
	if err != nil {
		t.Fatal(err)
	}

	blockType := cfgBlock.BlockType()
	if blockType != "provider" {
		t.Fatalf("Expected block type to be 'provider', given: %q",
			blockType)
	}
}

func TestParser_ParseBlockFromHcl_unknown(t *testing.T) {
	caps := lsp.TextDocumentClientCapabilities{}
	p := NewParser(emptyLogger(), caps)

	hclBlock := parseHclBlock(t, `meal "currywurst" {
}`)

	_, err := p.ParseBlockFromHcl(hclBlock)

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

func TestParser_ParseBlockFromHcl_error(t *testing.T) {
	caps := lsp.TextDocumentClientCapabilities{}
	p := NewParser(emptyLogger(), caps)

	hclBlock := parseHclBlock(t, `provider "currywurst" "extra" {
}`)

	_, err := p.ParseBlockFromHcl(hclBlock)

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
