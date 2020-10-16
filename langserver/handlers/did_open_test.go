package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/session"
)

func TestLangServer_didOpenWithoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"github\" {}",
			"uri": "%s/main.tf"
		}
	}`, TempDir(t).URI())}, session.SessionNotInitialized.Err())
}

func TestHumanReadablePath(t *testing.T) {
	fh := TempDir(t)

	err := os.Mkdir(filepath.Join(fh.Dir(), "testDir"), os.ModeDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := "testDir"
	path := humanReadablePath(fh.Dir(), "testDir")
	if path != expectedPath {
		t.Fatalf("expected %q, given: %q", expectedPath, path)
	}
}
