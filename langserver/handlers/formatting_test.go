package handlers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/session"
)

func TestLangServer_formattingWithoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"github\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, TempDir(t).URI())}, session.SessionNotInitialized.Err())
}

func TestLangServer_formatting_basic(t *testing.T) {
	tmpDir := TempDir(t)
	InitDir(t, tmpDir.Dir())
	queue := validTfMockCalls()
	queue.Q = append(queue.Q, &exec.MockItem{
		Args:   []string{"fmt", "-"},
		Stdout: "provider \"test\" {\n\n}\n",
	})
	ls := langserver.NewLangServerMock(t, NewMockSession(map[string]*rootmodule.RootModuleMock{
		tmpDir.Dir(): {TerraformExecQueue: queue},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDir(t).URI())})
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider  \"test\"   {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, TempDir(t).URI())})
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			}
		}`, TempDir(t).URI())}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": [
				{
					"range": {
						"start": { "line": 0, "character": 0 },
						"end": { "line": 1, "character": 0 }
					},
					"newText": "provider \"test\" {\n"
				}
			]
		}`)
}
