package lsp

import (
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/sourcegraph/go-lsp"
)

func TestLspRangeToHCL(t *testing.T) {
	testData := []struct {
		Name    string
		Content string
		Range   lsp.Range
		Expect  hcl.Range
	}{
		{
			Name:    "normal case",
			Content: "hello world",
			// the range part of "world"
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      0,
					Character: 6,
				},
				End: lsp.Position{
					Line:      0,
					Character: 11,
				},
			},
			Expect: hcl.Range{
				Start: hcl.Pos{
					Line:   1,
					Column: 7,
					Byte:   6,
				},
				End: hcl.Pos{
					Line:   1,
					Column: 12,
					Byte:   11,
				},
			},
		},
		{
			Name: "contain 𐐀",
			// 𐐀 in utf-16 has two unit
			// in utf-8 has four unit
			Content: "hello 𐐀a𐐀a world",
			// the range part of "a𐐀a"
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      0,
					Character: 8,
				},
				End: lsp.Position{
					Line:      0,
					Character: 12,
				},
			},
			Expect: hcl.Range{
				Start: hcl.Pos{
					Line:   1,
					Column: 9,
					Byte:   10,
				},
				End: hcl.Pos{
					Line:   1,
					Column: 13,
					Byte:   16,
				},
			},
		},
	}

	for _, v := range testData {
		t.Logf("[DEBUG] Testing %q", v.Name)

		result, err := lspRangeToHCL(v.Range, &file{
			fh:   FileHandlerFromDocumentURI(lsp.DocumentURI("file:///test.tf")),
			text: []byte(v.Content),
		})
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(v.Expect.Start, result.Start) {
			t.Fatalf("Expected %+v but got %+v", v.Expect.Start, result.Start)
		}

		if !reflect.DeepEqual(v.Expect.End, result.End) {
			t.Fatalf("Expected %+v but got %+v", v.Expect.End, result.End)
		}
	}
}

func TestHCLRangeToLSP(t *testing.T) {
	testData := []struct {
		Name   string
		Range  hcl.Range
		Expect lsp.Range
	}{
		{
			Name: "Range never less than 0",
			Range: hcl.Range{
				Start: hcl.Pos{
					Line:   -1,
					Column: -1,
				},
				End: hcl.Pos{
					Line:   -1,
					Column: -1,
				},
			},
			Expect: lsp.Range{
				Start: lsp.Position{
					Line:      0,
					Character: 0,
				},
				End: lsp.Position{
					Line:      0,
					Character: 0,
				},
			},
		},
	}

	for _, v := range testData {
		t.Logf("[DEBUG] Testing %q", v.Name)

		result := HCLRangeToLSP(v.Range)

		if !reflect.DeepEqual(v.Expect.Start, result.Start) {
			t.Fatalf("Expected %+v but got %+v", v.Expect.Start, result.Start)
		}

		if !reflect.DeepEqual(v.Expect.End, result.End) {
			t.Fatalf("Expected %+v but got %+v", v.Expect.End, result.End)
		}
	}
}
