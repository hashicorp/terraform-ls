package handlers

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/langserver"
)

func TestInitalizeAndShutdown(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(validTfMockCalls()))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDirUri())}, `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"capabilities": {
				"textDocumentSync": {
					"openClose": true,
					"change": 1
				},
				"completionProvider": {},
				"documentFormattingProvider":true
			}
		}
	}`)
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "shutdown", ReqParams: `{}`},
		`{
		"jsonrpc": "2.0",
		"id": 2,
		"result": null
	}`)
}

func TestEOF(t *testing.T) {
	ms := newMockSession(validTfMockCalls())
	ls := langserver.NewLangServerMock(t, ms.new)
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDirUri())}, `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"capabilities": {
				"textDocumentSync": {
					"openClose": true,
					"change": 1
				},
				"completionProvider": {},
				"documentFormattingProvider":true
			}
		}
	}`)

	ls.CloseClientStdout(t)

	if !ms.StopFuncCalled() {
		t.Fatal("Expected service to stop on EOF")
	}
	if ls.StopFuncCalled() {
		t.Fatal("Expected server not to stop on EOF")
	}
}

func validTfMockCalls() *exec.MockQueue {
	return &exec.MockQueue{
		Q: []*exec.MockItem{
			{
				Args:   []string{"version"},
				Stdout: "Terraform v0.12.0\n",
			},
			{
				Args:   []string{"providers", "schema", "-json"},
				Stdout: "{\"format_version\":\"0.1\"}\n",
			},
		},
	}
}

func TestMain(m *testing.M) {
	if v := os.Getenv("TF_LS_MOCK"); v != "" {
		os.Exit(exec.ExecuteMockData(v))
		return
	}

	os.Exit(m.Run())
}

func TempDirUri() string {
	return lsp.FileHandlerFromPath(os.TempDir()).URI()
}
