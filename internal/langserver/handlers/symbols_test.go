package handlers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_symbols_basic(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Dir())

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Dir(): validTfMockCalls(),
			},
		},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {
			"textDocument": {
				"documentSymbol": {
					"symbolKind": {
						"valueSet": [
							1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
							11, 12, 13, 14, 15, 16, 17, 18,
							19, 20, 21, 22, 23, 24, 25, 26
						]
					},
					"hierarchicalDocumentSymbolSupport": true,
					"tagSupport": {
						"valueSet": [ 1 ]
					},
					"labelSupport": true
				}
			}
		},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI())})
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
			"text": "provider \"github\" {}",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI())})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/documentSymbol",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI())}, `{
		"jsonrpc": "2.0",
		"id": 3,
		"result": [
			{
				"name": "provider \"github\"",
				"kind": 5,
				"range": {
					"start": {"line": 0, "character": 0},
					"end": {"line": 0, "character": 20}
				},
				"selectionRange": {
					"start": {"line": 0, "character": 0},
					"end": {"line": 0, "character": 20}
				}
			}
		]
	}`)
}
