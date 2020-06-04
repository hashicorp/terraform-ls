package handlers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/session"
)

func TestHover_withoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/hover",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 1,
				"line": 1
			}
		}`, TempDirUri())}, session.SessionNotInitialized.Err())
}

func TestHover_withValidData(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(&exec.MockQueue{
		Q: []*exec.MockItem{
			{
				Args:   []string{"version"},
				Stdout: "Terraform v0.12.0\n",
			},
			{
				Args:   []string{"providers", "schema", "-json"},
				Stdout: testSchemaOutput,
			},
		},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDirUri())})
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
			"text": "provider \"test\" {\nbase_url = \"\"\nsingle_block {\nnested_string = \"foo\"\n}\n}\n",
			"uri": "%s/main.tf"
		}
	}`, TempDirUri())})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/hover",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 1,
				"line": 1
			}
		}`, TempDirUri())}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"contents": ["**base_url**\n_Optional, string_\n\nDesc **2**"]
			}
		}`)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/hover",
		ReqParams: fmt.Sprintf(`{
				"textDocument": {
					"uri": "%s/main.tf"
				},
				"position": {
					"character": 1,
					"line": 2
				}
			}`, TempDirUri())}, `{
				"jsonrpc": "2.0",
				"id": 4,
				"result": {
					"contents": ["**single_block**\n_Block, single_\n\nSingle Block"]
				}
			}`)
}
