package lang

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
)

func TestParser_BlockTypeCandidates_len(t *testing.T) {
	p := newParser()

	content := `
provider "aws" {
}`
	pos := hcl.Pos{
		Line:   1,
		Column: 1,
		Byte:   0,
	}
	tFile := ihcl.NewTestFile([]byte(content))
	candidates := p.BlockTypeCandidates(tFile, pos)
	if candidates.Len() < 3 {
		t.Fatalf("Expected >= 3 candidates, %d given", candidates.Len())
	}
}

func TestParser_BlockTypeCandidates_snippet(t *testing.T) {
	p := newParser()

	content := `
provider "aws" {
}`
	pos := hcl.Pos{
		Line:   1,
		Column: 1,
		Byte:   0,
	}
	tFile := ihcl.NewTestFile([]byte(content))
	list := p.BlockTypeCandidates(tFile, pos)
	rendered := renderCandidates(list, hcl.InitialPos)
	sortRenderedCandidates(rendered)

	expectedCandidate := renderedCandidate{
		Label:  "data",
		Detail: "",
		Documentation: "A data block requests that Terraform read from a given data source and export the result " +
			"under the given local name. The name is used to refer to this resource from elsewhere in the same " +
			"Terraform module, but has no significance outside of the scope of a module.",
		Snippet: renderedSnippet{
			Pos: hcl.InitialPos,
			Text: `data "${1}" "${2:name}" {
  ${3}
}`,
		},
	}
	if diff := cmp.Diff(expectedCandidate, rendered[0]); diff != "" {
		t.Fatalf("Completion candidate does not match.\n%s", diff)
	}
}

func TestParser_ParseBlockFromTokens(t *testing.T) {
	testCases := []struct {
		name string
		cfg  string

		expectedBlockType string
		expectedErr       error
	}{
		{
			"valid",
			`provider "currywurst" {
}`,
			"provider",
			nil,
		},
		{
			"unknown block",
			`meal "currywurst" {
}`,
			"",
			&unknownBlockTypeErr{"meal"},
		},
		{
			"error from factory",
			`provider "currywurst" "extra" {
}`,
			"provider",
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			tBlock := newTestBlock(t, tc.cfg)

			p := newParser()
			cfgBlock, err := p.ParseBlockFromTokens(tBlock)
			if err != nil {
				if errors.Is(err, tc.expectedErr) {
					return
				}
				t.Fatalf("Error doesn't match.\nexpected: %v\ngiven: %v\n",
					tc.expectedErr, err.Error())
			}
			if tc.expectedErr != nil {
				t.Fatalf("Expected error: %s", tc.expectedErr)
			}

			blockType := cfgBlock.BlockType()
			if blockType != tc.expectedBlockType {
				t.Fatalf("Block type doesn't match.\nexpected: %q\ngiven: %q\n",
					tc.expectedBlockType, blockType)
			}
		})
	}
}

func newTestBlock(t *testing.T, src string) ihcl.TokenizedBlock {
	t.Helper()
	b, err := ihcl.NewTestBlock([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func parseHclBlock(t *testing.T, src string) *hcl.Block {
	f, diags := hclsyntax.ParseConfig([]byte(src), "/test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	return f.OutermostBlockAtPos(hcl.InitialPos)
}

func parseHclSyntaxBlocks(t *testing.T, src string) hclsyntax.Blocks {
	f, diags := hclsyntax.ParseConfig([]byte(src), "/test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	body, ok := f.Body.(*hclsyntax.Body)
	if !ok {
		t.Fatalf("unsupported configuration format: %T", f.Body)
	}

	return body.Blocks
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}
