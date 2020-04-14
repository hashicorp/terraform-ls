package handlers

import (
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/session"
)

func TestLangServer_formattingWithoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: `{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"github\" {\n\n}\n",
			"uri": "file:///var/main.tf"
		}
	}`}, session.SessionNotInitialized.Err())
}

func TestLangServer_formatting(t *testing.T) {
	queue := validTfMockCalls()
	queue.Q = append(queue.Q, &exec.MockItem{
		Args:   []string{"fmt", "-"},
		Stdout: "provider \"test\" {\n\n}\n",
	})
	ls := langserver.NewLangServerMock(t, NewMock(queue))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: `{
	    "capabilities": {},
	    "rootUri": "file:///tmp",
	    "processId": 12345
	}`})
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: `{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider  \"test\"   {\n\n}\n",
			"uri": "file:///tmp/main.tf"
		}
	}`})
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: `{
			"textDocument": {
				"uri": "file:///tmp/main.tf"
			}
		}`}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": [
				{
					"range": {
						"start": { "line": 0, "character": 0 },
						"end": { "line": 3, "character": 0 }
					},
					"newText": "provider \"test\" {\n\n}\n"
				}
			]
		}`)
}
