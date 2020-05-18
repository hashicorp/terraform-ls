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
		text: "something else",
	}
	err := f.applyChange(fChange)
	if err != nil {
		t.Fatal(err)
	}
}

type fileChange struct {
	text string
}

func (fc *fileChange) Text() string {
	return fc.text
}

func (fc *fileChange) Range() hcl.Range {
	return hcl.Range{
		// TODO: Implement partial updates
	}
}
