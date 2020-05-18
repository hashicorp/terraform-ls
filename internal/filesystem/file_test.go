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
	f := NewFile("file:///test.tf", []byte("hello world"))

	fChange := &fileChange{
		text: "terraform",
		rng: hcl.Range{
			Start: hcl.Pos{
				Line:   1,
				Column: 6,
				Byte:   6,
			},
			End: hcl.Pos{
				Line:   1,
				Column: 11,
				Byte:   11,
			},
		},
	}
	err := f.applyChange(fChange)
	if err != nil {
		t.Fatal(err)
	}

	expected := "hello terraform"
	if string(f.content) != expected {
		t.Fatalf("expected: %q but actually: %q", expected, string(f.content))
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
