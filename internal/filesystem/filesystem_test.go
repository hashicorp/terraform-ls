package filesystem

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFilesystem_Change_notOpen(t *testing.T) {
	fs := testDocumentStorage()

	var changes DocumentChanges
	changes = append(changes, &testChange{})
	h := &testHandler{"file:///doesnotexist"}

	err := fs.ChangeDocument(h, changes)

	expectedErr := &UnknownDocumentErr{h}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestFilesystem_Change_closed(t *testing.T) {
	fs := testDocumentStorage()

	fh := &testHandler{"file:///doesnotexist"}
	fs.CreateAndOpenDocument(fh, []byte{})
	err := fs.CloseAndRemoveDocument(fh)
	if err != nil {
		t.Fatal(err)
	}

	var changes DocumentChanges
	changes = append(changes, &testChange{})
	err = fs.ChangeDocument(fh, changes)

	expectedErr := &UnknownDocumentErr{fh}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestFilesystem_Remove_unknown(t *testing.T) {
	fs := testDocumentStorage()

	fh := &testHandler{"file:///doesnotexist"}
	fs.CreateAndOpenDocument(fh, []byte{})
	err := fs.CloseAndRemoveDocument(fh)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.CloseAndRemoveDocument(fh)

	expectedErr := &UnknownDocumentErr{fh}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestFilesystem_Close_closed(t *testing.T) {
	fs := testDocumentStorage()

	fh := &testHandler{"file:///isnotopen"}
	fs.CreateDocument(fh, []byte{})
	err := fs.CloseAndRemoveDocument(fh)
	expectedErr := &DocumentNotOpenErr{fh}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestFilesystem_Change_noChanges(t *testing.T) {
	fs := testDocumentStorage()

	fh := &testHandler{"file:///test.tf"}
	fs.CreateAndOpenDocument(fh, []byte{})

	var changes DocumentChanges
	err := fs.ChangeDocument(fh, changes)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilesystem_Change_multipleChanges(t *testing.T) {
	fs := testDocumentStorage()

	fh := &testHandler{"file:///test.tf"}
	fs.CreateAndOpenDocument(fh, []byte{})

	var changes DocumentChanges
	changes = append(changes, &testChange{text: "ahoy"})
	changes = append(changes, &testChange{text: ""})
	changes = append(changes, &testChange{text: "quick brown fox jumped over\nthe lazy dog"})
	changes = append(changes, &testChange{text: "bye"})

	err := fs.ChangeDocument(fh, changes)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilesystem_GetDocument_success(t *testing.T) {
	fs := testDocumentStorage()

	dh := &testHandler{"file:///test.tf"}
	err := fs.CreateAndOpenDocument(dh, []byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}

	f, err := fs.GetDocument(dh)
	if err != nil {
		t.Fatal(err)
	}

	b := []byte("hello world")
	meta := NewDocumentMetadata(dh, b)
	meta.isOpen = true
	expectedFile := testDocument(t, dh, meta, b)
	if diff := cmp.Diff(expectedFile, f); diff != "" {
		t.Fatalf("File doesn't match: %s", diff)
	}
}

func TestFilesystem_GetDocument_unknownDocument(t *testing.T) {
	fs := testDocumentStorage()

	fh := &testHandler{"file:///test.tf"}
	_, err := fs.GetDocument(fh)

	expectedErr := &UnknownDocumentErr{fh}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

type testHandler struct {
	uri string
}

func (fh *testHandler) URI() string {
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

func testDocumentStorage() DocumentStorage {
	fs := NewFilesystem()
	fs.logger = testLogger()
	return fs
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}
