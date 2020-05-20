package filesystem

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestFile_ApplyChange_fullUpdate(t *testing.T) {
	f := NewFile("file:///test.tf", []byte("hello world"))

	fChange := &fileChange{
		text: "something else",
	}
	err := f.applyChange(fChange)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFile_ApplyChange_partialUpdate(t *testing.T) {
	testData := []struct {
		Name       string
		Content    string
		FileChange *fileChange
		Expect     string
	}{
		{
			Name:    "length grow: 4",
			Content: "hello world",
			FileChange: &fileChange{
				text: "terraform",
				rng: hcl.Range{
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
			Expect: "hello terraform",
		},
		{
			Name:    "length the same",
			Content: "hello world",
			FileChange: &fileChange{
				text: "earth",
				rng: hcl.Range{
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
			Expect: "hello earth",
		},
		{
			Name:    "length grow: -2",
			Content: "hello world",
			FileChange: &fileChange{
				text: "HCL",
				rng: hcl.Range{
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
			Expect: "hello HCL",
		},
		{
			Name:    "add utf-18 character",
			Content: "hello world",
			FileChange: &fileChange{
				text: "𐐀𐐀 ",
				rng: hcl.Range{
					Start: hcl.Pos{
						Line:   1,
						Column: 7,
						Byte:   6,
					},
					End: hcl.Pos{
						Line:   1,
						Column: 7,
						Byte:   6,
					},
				},
			},
			Expect: "hello 𐐀𐐀 world",
		},
		{
			Name:    "modify when containing utf-18 character",
			Content: "hello 𐐀𐐀 world",
			FileChange: &fileChange{
				text: "aa𐐀",
				rng: hcl.Range{
					Start: hcl.Pos{
						Line:   1,
						Column: 9,
						Byte:   10,
					},
					End: hcl.Pos{
						Line:   1,
						Column: 11,
						Byte:   14,
					},
				},
			},
			Expect: "hello 𐐀aa𐐀 world",
		},
	}

	for _, v := range testData {
		t.Logf("[DEBUG] Testing %q", v.Name)

		f := NewFile("file:///test.tf", []byte(v.Content))
		err := f.applyChange(v.FileChange)
		if err != nil {
			t.Fatal(err)
		}

		if string(f.content) != v.Expect {
			t.Fatalf("expected: %q but actually: %q", v.Expect, string(f.content))
		}
	}
}

type fileChange struct {
	text string
	rng  hcl.Range
}

func (fc *fileChange) Text() string {
	return fc.text
}

func (fc *fileChange) Range() hcl.Range {
	return fc.rng
}
