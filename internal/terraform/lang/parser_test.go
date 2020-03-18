package lang

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestParser_ParseBlockFromHCL(t *testing.T) {
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
			"",
			&invalidLabelsErr{
				BlockType: "provider",
				Labels:    []string{"currywurst", "extra"},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			hclBlock := parseHclBlock(t, tc.cfg)

			p := newParser()
			cfgBlock, err := p.ParseBlockFromHCL(hclBlock)
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

func parseHclBlock(t *testing.T, src string) *hcl.Block {
	f, diags := hclsyntax.ParseConfig([]byte(src), "/test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	return f.OutermostBlockAtPos(hcl.InitialPos)
}
