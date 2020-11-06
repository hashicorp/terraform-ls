package filesystem

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFilesystem_Change_notOpen(t *testing.T) {
	fs := testDocumentStorage()

	var changes DocumentChanges
	changes = append(changes, &testChange{})
	h := &testHandler{uri: "file:///doesnotexist"}

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

	fh := &testHandler{uri: "file:///doesnotexist"}
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

	fh := &testHandler{uri: "file:///doesnotexist"}
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

	fh := &testHandler{uri: "file:///isnotopen"}
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

	fh := &testHandler{uri: "file:///test.tf"}
	fs.CreateAndOpenDocument(fh, []byte{})

	var changes DocumentChanges
	err := fs.ChangeDocument(fh, changes)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilesystem_Change_multipleChanges(t *testing.T) {
	fs := testDocumentStorage()

	fh := &testHandler{uri: "file:///test.tf"}
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

	dh := &testHandler{uri: "file:///test.tf"}
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

	fh := &testHandler{uri: "file:///test.tf"}
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

func TestFilesystem_ReadFile_osOnly(t *testing.T) {
	tmpDir := TempDir(t)
	f, err := os.Create(filepath.Join(tmpDir, "testfile"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	content := "lorem ipsum"
	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}

	fs := NewFilesystem()
	b, err := fs.ReadFile(filepath.Join(tmpDir, "testfile"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != content {
		t.Fatalf("expected content to match %q, given: %q",
			content, string(b))
	}

	_, err = fs.ReadFile(filepath.Join(tmpDir, "not-existing"))
	if err == nil {
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_ReadFile_memOnly(t *testing.T) {
	fs := NewFilesystem()
	fh := &testHandler{uri: "file:///tmp/test.tf"}
	content := "test content"
	err := fs.CreateDocument(fh, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	b, err := fs.ReadFile(fh.FullPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != content {
		t.Fatalf("expected content to match %q, given: %q",
			content, string(b))
	}

	_, err = fs.ReadFile(filepath.Join("tmp", "not-existing"))
	if err == nil {
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_ReadFile_memAndOs(t *testing.T) {
	tmpDir := TempDir(t)
	testPath := filepath.Join(tmpDir, "testfile")

	f, err := os.Create(testPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	osContent := "os content"
	_, err = f.WriteString(osContent)
	if err != nil {
		t.Fatal(err)
	}

	fs := NewFilesystem()

	fh := testHandlerFromPath(testPath)
	memContent := "in-mem content"
	err = fs.CreateDocument(fh, []byte(memContent))
	if err != nil {
		t.Fatal(err)
	}

	b, err := fs.ReadFile(fh.FullPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != memContent {
		t.Fatalf("expected content to match %q, given: %q",
			memContent, string(b))
	}

	_, err = fs.ReadFile(filepath.Join(tmpDir, "not-existing"))
	if err == nil {
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_ReadDir(t *testing.T) {
	tmpDir := TempDir(t)

	f, err := os.Create(filepath.Join(tmpDir, "osfile"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	fs := NewFilesystem()

	fh := testHandlerFromPath(filepath.Join(tmpDir, "memfile"))
	err = fs.CreateDocument(fh, []byte("test"))
	if err != nil {
		t.Fatal(err)
	}

	fis, err := fs.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedFis := []string{"memfile", "osfile"}
	names := namesFromFileInfos(fis)
	if diff := cmp.Diff(expectedFis, names); diff != "" {
		t.Fatalf("file list mismatch: %s", diff)
	}
}

func namesFromFileInfos(fis []os.FileInfo) []string {
	names := make([]string, len(fis), len(fis))
	for i, fi := range fis {
		names[i] = fi.Name()
	}
	return names
}

func TestFilesystem_Open_osOnly(t *testing.T) {
	tmpDir := TempDir(t)
	f, err := os.Create(filepath.Join(tmpDir, "testfile"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	content := "lorem ipsum"
	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}

	fs := NewFilesystem()
	f1, err := fs.Open(filepath.Join(tmpDir, "testfile"))
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	f2, err := fs.Open(filepath.Join(tmpDir, "not-existing"))
	if err == nil {
		defer f2.Close()
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_Open_memOnly(t *testing.T) {
	fs := NewFilesystem()
	tmpDir := TempDir(t)
	testPath := filepath.Join(tmpDir, "test.tf")
	fh := testHandlerFromPath(testPath)

	content := "test content"
	err := fs.CreateDocument(fh, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	f1, err := fs.Open(fh.FullPath())
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	f2, err := fs.Open(filepath.Join("tmp", "not-existing"))
	if err == nil {
		defer f2.Close()
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_Open_memAndOs(t *testing.T) {
	tmpDir := TempDir(t)
	testPath := filepath.Join(tmpDir, "testfile")

	f, err := os.Create(testPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	osContent := "os content"
	_, err = f.WriteString(osContent)
	if err != nil {
		t.Fatal(err)
	}

	fs := NewFilesystem()

	fh := testHandlerFromPath(testPath)
	memContent := "in-mem content"
	err = fs.CreateDocument(fh, []byte(memContent))
	if err != nil {
		t.Fatal(err)
	}

	f1, err := fs.Open(fh.FullPath())
	if err != nil {
		t.Fatal(err)
	}
	fi, err := f1.Stat()
	if err != nil {
		t.Fatal(err)
	}
	size := int(fi.Size())
	if size != len(memContent) {
		t.Fatalf("expected size to match %d, given: %d",
			len(memContent), size)
	}

	_, err = fs.Open(filepath.Join(tmpDir, "not-existing"))
	if err == nil {
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_Create_memOnly(t *testing.T) {
	fs := NewFilesystem()
	tmpDir := TempDir(t)
	testPath := filepath.Join(tmpDir, "test.tf")
	fh := testHandlerFromPath(testPath)

	content := "test content"
	err := fs.CreateDocument(fh, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	infos, err := fs.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedFis := []string{"test.tf"}
	names := namesFromFileInfos(infos)
	if diff := cmp.Diff(expectedFis, names); diff != "" {
		t.Fatalf("file list mismatch: %s", diff)
	}
}

func TempDir(t *testing.T) string {
	tmpDir := filepath.Join(os.TempDir(), "terraform-ls", t.Name())

	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		if os.IsExist(err) {
			return tmpDir
		}
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
	})

	return tmpDir
}

func testHandlerFromPath(path string) DocumentHandler {
	return &testHandler{uri: URIFromPath(path), fullPath: path}
}

type testHandler struct {
	uri      string
	fullPath string
}

func (fh *testHandler) URI() string {
	return fh.uri
}

func (fh *testHandler) FullPath() string {
	return fh.fullPath
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
