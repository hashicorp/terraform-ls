package filesystem

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/spf13/afero"
)

func TestFile_ApplyChange_fullUpdate(t *testing.T) {
	fs := testDocumentStorage()
	dh := &testHandler{"file:///test.tf"}

	err := fs.CreateAndOpenDocument(dh, []byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}

	changes := []DocumentChange{
		&testChange{text: "something else"},
	}
	err = fs.ChangeDocument(dh, changes)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := fs.GetDocument(dh)
	if err != nil {
		t.Fatal(err)
	}
	given, err := doc.Text()
	if err != nil {
		t.Fatal(err)
	}

	expectedText := "something else"
	if diff := cmp.Diff(expectedText, string(given)); diff != "" {
		t.Fatalf("content mismatch: %s", diff)
	}
}

func TestFile_ApplyChange_partialUpdate(t *testing.T) {
	testData := []struct {
		Name       string
		Content    string
		FileChange *testChange
		Expect     string
	}{
		{
			Name:    "length grow: 4",
			Content: "hello world",
			FileChange: &testChange{
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
			FileChange: &testChange{
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
			FileChange: &testChange{
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
			Name:    "zero-length range",
			Content: "hello world",
			FileChange: &testChange{
				text: "abc ",
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
			Expect: "hello abc world",
		},
		{
			Name:    "add utf-18 character",
			Content: "hello world",
			FileChange: &testChange{
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
			FileChange: &testChange{
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
		fs := testDocumentStorage()
		dh := &testHandler{"file:///test.tf"}

		err := fs.CreateAndOpenDocument(dh, []byte(v.Content))
		if err != nil {
			t.Fatal(err)
		}

		changes := []DocumentChange{v.FileChange}
		err = fs.ChangeDocument(dh, changes)
		if err != nil {
			t.Fatal(err)
		}

		doc, err := fs.GetDocument(dh)
		if err != nil {
			t.Fatal(err)
		}

		text, err := doc.Text()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(v.Expect, string(text)); diff != "" {
			t.Fatalf("%s: content mismatch: %s", v.Name, diff)
		}
	}
}

func testDocument(t *testing.T, dh DocumentHandler, meta *documentMetadata, b []byte) Document {
	fs := afero.NewMemMapFs()
	f, err := fs.Create(dh.FullPath())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = f.Write(b)
	if err != nil {
		t.Fatal(err)
	}

	return &document{
		meta: meta,
		fo:   fs,
	}
}

type testChange struct {
	text string
	rng  hcl.Range
}

func (fc *testChange) Text() string {
	return fc.text
}

func (fc *testChange) Range() hcl.Range {
	return fc.rng
}
