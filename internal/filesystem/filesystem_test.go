package filesystem

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/source"
)

func TestFilesystem_Change_notOpen(t *testing.T) {
	fs := NewFilesystem()

	var changes FileChanges
	changes = append(changes, &testChange{})
	h := &testHandler{"file:///doesnotexist"}

	err := fs.Change(h, changes)

	expectedErr := &FileNotOpenErr{h}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestFilesystem_Change_closed(t *testing.T) {
	fs := NewFilesystem()

	fh := &testHandler{"file:///doesnotexist"}
	fs.Open(&testFile{
		testHandler: fh,
		text:        "",
	})
	err := fs.Close(fh)
	if err != nil {
		t.Fatal(err)
	}

	var changes FileChanges
	changes = append(changes, &testChange{})
	err = fs.Change(fh, changes)

	expectedErr := &FileNotOpenErr{fh}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestFilesystem_Close_closed(t *testing.T) {
	fs := NewFilesystem()

	fh := &testHandler{"file:///doesnotexist"}
	fs.Open(&testFile{
		testHandler: fh,
		text:        "",
	})
	err := fs.Close(fh)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.Close(fh)

	expectedErr := &FileNotOpenErr{fh}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestFilesystem_Change_noChanges(t *testing.T) {
	fs := NewFilesystem()

	fh := &testHandler{"file:///test.tf"}
	fs.Open(&testFile{
		testHandler: fh,
		text:        "",
	})

	var changes FileChanges
	err := fs.Change(fh, changes)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilesystem_Change_multipleChanges(t *testing.T) {
	fs := NewFilesystem()

	fh := &testHandler{"file:///test.tf"}
	fs.Open(&testFile{
		testHandler: fh,
		text:        "",
	})

	var changes FileChanges
	changes = append(changes, &testChange{text: "ahoy"})
	changes = append(changes, &testChange{text: ""})
	changes = append(changes, &testChange{text: "quick brown fox jumped over\nthe lazy dog"})
	changes = append(changes, &testChange{text: "bye"})

	err := fs.Change(fh, changes)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilesystem_GetFile_success(t *testing.T) {
	fs := NewFilesystem()

	fh := &testHandler{"file:///test.tf"}
	err := fs.Open(&testFile{
		testHandler: fh,
		text:        "hello world",
	})
	if err != nil {
		t.Fatal(err)
	}

	f, err := fs.GetFile(fh)
	if err != nil {
		t.Fatal(err)
	}

	expectedFile := &file{
		content: []byte("hello world"),
		open:    true,
	}
	opts := []cmp.Option{
		cmp.AllowUnexported(file{}),
	}
	if diff := cmp.Diff(expectedFile, f, opts...); diff != "" {
		t.Fatalf("File doesn't match: %s", diff)
	}
}

func TestFilesystem_GetFile_unopenedFile(t *testing.T) {
	fs := NewFilesystem()

	fh := &testHandler{"file:///test.tf"}
	_, err := fs.GetFile(fh)

	expectedErr := &FileNotOpenErr{fh}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

type testFile struct {
	*testHandler
	text string
}

func (f *testFile) Text() []byte {
	return []byte(f.text)
}

func (f *testFile) Lines() source.Lines {
	return source.Lines{}
}

type testHandler struct {
	uri string
}

func (fh *testHandler) DocumentURI() string {
	return fh.uri
}

func (fh *testHandler) FullPath() string {
	return ""
}

func (fh *testHandler) Dir() string {
	return ""
}

func (fh *testHandler) Filename() string {
	return ""
}
func (fh *testHandler) Version() int {
	return 0
}

type testChange struct {
	text string
}

func (ch *testChange) Text() string {
	return ch.text
}
