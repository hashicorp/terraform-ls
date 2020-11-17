package filesystem

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

func TestFile_ApplyChange_fullUpdate(t *testing.T) {
	fs := testDocumentStorage()
	dh := &testHandler{uri: "file:///test.tf"}

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
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 11,
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
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 11,
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
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 11,
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
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 6,
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
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 6,
					},
					End: Pos{
						Line:   0,
						Column: 6,
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
				rng: &Range{
					Start: Pos{
						Line:   0,
						Column: 8,
					},
					End: Pos{
						Line:   0,
						Column: 10,
					},
				},
			},
			Expect: "hello 𐐀aa𐐀 world",
		},
	}

	for _, v := range testData {
		fs := testDocumentStorage()
		dh := &testHandler{uri: "file:///test.tf"}

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

func TestFile_ApplyChange_partialUpdateMultipleChanges(t *testing.T) {
	testData := []struct {
		Content     string
		FileChanges DocumentChanges
		Expect      string
	}{
		{
			Content: `variable "service_host" {
  default = "blah"
}

module "app" {
  source = "./sub"
  service_listeners = [
    {
      hosts    = [var.service_host]
      listener = ""
    }
  ]
}
`,
			FileChanges: DocumentChanges{
				&testChange{
					text: "\n",
					rng: &Range{
						Start: Pos{Line: 8, Column: 18},
						End:   Pos{Line: 8, Column: 18},
					},
				},
				&testChange{
					text: "      ",
					rng: &Range{
						Start: Pos{Line: 9, Column: 0},
						End:   Pos{Line: 9, Column: 0},
					},
				},
				&testChange{
					text: "  ",
					rng: &Range{
						Start: Pos{Line: 9, Column: 6},
						End:   Pos{Line: 9, Column: 6},
					},
				},
			},
			Expect: `variable "service_host" {
  default = "blah"
}

module "app" {
  source = "./sub"
  service_listeners = [
    {
      hosts    = [
        var.service_host]
      listener = ""
    }
  ]
}
`,
		},
	}

	for _, v := range testData {
		fs := testDocumentStorage()
		dh := &testHandler{uri: "file:///test.tf"}

		err := fs.CreateAndOpenDocument(dh, []byte(v.Content))
		if err != nil {
			t.Fatal(err)
		}

		changes := v.FileChanges
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
			t.Fatalf("content mismatch: %s", diff)
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
		fs:   fs,
	}
}

type testChange struct {
	text string
	rng  *Range
}

func (fc *testChange) Text() string {
	return fc.text
}

func (fc *testChange) Range() *Range {
	return fc.rng
}
