package state

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/source"
)

func TestDocumentStore_UpdateDocument_notFound(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testHandle := document.HandleFromURI("file:///not/found.tf")
	err = s.DocumentStore.UpdateDocument(testHandle, []byte{}, 2)
	expectedErr := &document.DocumentNotFound{URI: testHandle.FullURI()}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestDocumentStore_CloseDocument_notFound(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testHandle := document.HandleFromURI("file:///not/found.tf")
	err = s.DocumentStore.CloseDocument(testHandle)

	expectedErr := &document.DocumentNotFound{URI: testHandle.FullURI()}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestDocumentStore_UpdateDocument_emptyText(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testHandle := document.HandleFromURI("file:///dir/test.tf")

	err = s.DocumentStore.OpenDocument(testHandle, "terraform", 0, []byte("foo"))
	if err != nil {
		t.Fatal(err)
	}

	err = s.DocumentStore.UpdateDocument(testHandle, []byte{}, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDocumentStore_UpdateDocument_basic(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testHandle := document.HandleFromURI("file:///dir/test.tf")

	err = s.DocumentStore.OpenDocument(testHandle, "terraform", 0, []byte("foo"))
	if err != nil {
		t.Fatal(err)
	}

	err = s.DocumentStore.UpdateDocument(testHandle, []byte("barx"), 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDocumentStore_GetDocument_basic(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	s.DocumentStore.TimeProvider = testTimeProvider

	testHandle := document.HandleFromURI("file:///dir/test.tf")
	err = s.DocumentStore.OpenDocument(testHandle, "terraform", 0, []byte("foobar"))
	if err != nil {
		t.Fatal(err)
	}

	doc, err := s.DocumentStore.GetDocument(testHandle)
	if err != nil {
		t.Fatal(err)
	}

	text := []byte("foobar")
	expectedDocument := &document.Document{
		Dir:        testHandle.Dir,
		Filename:   testHandle.Filename,
		ModTime:    testTimeProvider(),
		LanguageID: "terraform",
		Version:    0,
		Text:       text,
		Lines:      source.MakeSourceLines(testHandle.Filename, text),
	}
	if diff := cmp.Diff(expectedDocument, doc); diff != "" {
		t.Fatalf("File doesn't match: %s", diff)
	}
}

func TestDocumentStore_GetDocument_notFound(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	testHandle := document.HandleFromURI("file:///not/found.tf")
	_, err = s.DocumentStore.GetDocument(testHandle)

	expectedErr := &document.DocumentNotFound{URI: testHandle.FullURI()}
	if err == nil {
		t.Fatalf("Expected error: %s", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("Unexpected error.\nexpected: %#v\ngiven: %#v",
			expectedErr, err)
	}
}

func TestDocumentStore_ListDocumentsInDir(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	s.DocumentStore.TimeProvider = testTimeProvider

	testHandle1 := document.HandleFromURI("file:///dir/test1.tf")
	err = s.DocumentStore.OpenDocument(testHandle1, "terraform", 0, []byte("foobar"))
	if err != nil {
		t.Fatal(err)
	}

	testHandle2 := document.HandleFromURI("file:///dir/test2.tf")
	err = s.DocumentStore.OpenDocument(testHandle2, "terraform", 0, []byte("foobar"))
	if err != nil {
		t.Fatal(err)
	}

	dirHandle := document.DirHandleFromURI("file:///dir")
	docs, err := s.DocumentStore.ListDocumentsInDir(dirHandle)
	if err != nil {
		t.Fatal(err)
	}

	expectedDocs := []*document.Document{
		{
			Dir:        dirHandle,
			Filename:   "test1.tf",
			ModTime:    testTimeProvider(),
			LanguageID: "terraform",
			Version:    0,
			Text:       []byte("foobar"),
			Lines:      source.MakeSourceLines("test1.tf", []byte("foobar")),
		},
		{
			Dir:        dirHandle,
			Filename:   "test2.tf",
			ModTime:    testTimeProvider(),
			LanguageID: "terraform",
			Version:    0,
			Text:       []byte("foobar"),
			Lines:      source.MakeSourceLines("test2.tf", []byte("foobar")),
		},
	}
	if diff := cmp.Diff(expectedDocs, docs); diff != "" {
		t.Fatalf("unexpected docs: %s", diff)
	}
}

func TestDocumentStore_IsDocumentOpen(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	s.DocumentStore.TimeProvider = testTimeProvider

	testHandle1 := document.HandleFromURI("file:///dir/test1.tf")
	err = s.DocumentStore.OpenDocument(testHandle1, "terraform", 0, []byte("foobar"))
	if err != nil {
		t.Fatal(err)
	}

	isOpen, err := s.DocumentStore.IsDocumentOpen(testHandle1)
	if err != nil {
		t.Fatal(err)
	}
	if !isOpen {
		t.Fatal("expected first document to be open")
	}

	testHandle2 := document.HandleFromURI("file:///dir/test2.tf")
	isOpen, err = s.DocumentStore.IsDocumentOpen(testHandle2)
	if err != nil {
		t.Fatal(err)
	}
	if isOpen {
		t.Fatal("expected second document not to be open")
	}
}

func TestDocumentStore_HasOpenDocuments(t *testing.T) {
	s, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	s.DocumentStore.TimeProvider = testTimeProvider

	testHandle1 := document.HandleFromURI("file:///dir/test1.tf")
	err = s.DocumentStore.OpenDocument(testHandle1, "terraform", 0, []byte("foobar"))
	if err != nil {
		t.Fatal(err)
	}

	dirHandle := document.DirHandleFromURI("file:///dir")
	hasOpenDocs, err := s.DocumentStore.HasOpenDocuments(dirHandle)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOpenDocs {
		t.Fatal("expected to find open documents")
	}

	secondDirHandle := document.DirHandleFromURI("file:///dir-x")
	hasOpenDocs, err = s.DocumentStore.HasOpenDocuments(secondDirHandle)
	if err != nil {
		t.Fatal(err)
	}
	if hasOpenDocs {
		t.Fatal("expected to find no open documents")
	}

}

func testTimeProvider() time.Time {
	return time.Date(2017, 1, 16, 0, 0, 0, 0, time.UTC)
}
