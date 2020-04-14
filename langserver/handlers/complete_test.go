package handlers

import (
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/session"
)

func TestCompletion_withoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: `{
			"textDocument": {
				"uri": "file:///var/main.tf"
			},
			"position": {
				"character": 0,
				"line": 1
			}
		}`}, session.SessionNotInitialized.Err())
}

func TestCompletion_withValidData(t *testing.T) {
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
			"text": "provider \"test\" {\n\n}\n",
			"uri": "file:///tmp/main.tf"
		}
	}`})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: `{
			"textDocument": {
				"uri": "file:///tmp/main.tf"
			},
			"position": {
				"character": 0,
				"line": 1
			}
		}`}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label":"anonymous",
						"kind":5,
						"detail":"(Optional, number) Desc 1",
						"insertTextFormat":1
					},
					{
						"label":"base_url",
						"kind":5,
						"detail":"(Optional, string) Desc 2",
						"insertTextFormat":1
					},
					{
						"label":"individual",
						"kind":5,
						"detail":"(Optional, bool) Desc 3",
						"insertTextFormat":1
					}
				]
			}
		}`)
}

var testSchemaOutput = `{
  "format_version": "0.1",
  "provider_schemas": {
    "test": {
      "provider": {
        "version": 0,
        "block": {
          "attributes": {
            "anonymous": {
              "type": "number",
              "description": "Desc 1",
              "optional": true
            },
            "base_url": {
              "type": "string",
              "description": "Desc 2",
              "optional": true
            },
            "individual": {
              "type": "bool",
              "description": "Desc 3",
              "optional": true
            }
          }
        }
      }
    }
  }
}`
