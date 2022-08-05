package handlers

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
	"github.com/stretchr/testify/mock"
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
	}`, TempDir(t).URI)}, session.SessionNotInitialized.Err())
}

func TestLangServer_didOpenLanguageIdStored(t *testing.T) {
	tmpDir := TempDir(t)

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): validTfMockCalls(),
			},
		},
		StateStore:      ss,
		WalkerCollector: wc,
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})

	originalText := `variable "service_host" {
  default = "blah"
}
`
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
    "textDocument": {
        "languageId": "terraform",
        "version": 0,
        "uri": "%s/main.tf",
        "text": %q
    }
}`, TempDir(t).URI, originalText)})
	waitForAllJobs(t, ss)

	path := filepath.Join(TempDir(t).Path(), "main.tf")
	dh := document.HandleFromPath(path)
	doc, err := ss.DocumentStore.GetDocument(dh)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(doc.LanguageID, string("terraform")); diff != "" {
		t.Fatalf("unexpected languageID: %s", diff)
	}
	fullPath := doc.FullPath()
	if diff := cmp.Diff(fullPath, string(path)); diff != "" {
		t.Fatalf("unexpected fullPath: %s", diff)
	}
	version := doc.Version
	if diff := cmp.Diff(version, int(0)); diff != "" {
		t.Fatalf("unexpected version: %s", diff)
	}
}
