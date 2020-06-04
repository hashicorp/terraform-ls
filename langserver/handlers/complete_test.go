package handlers

import (
	"fmt"
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
		ReqParams: fmt.Sprintf(`{
            "textDocument": {
                "uri": "%s/main.tf"
            },
            "position": {
                "character": 0,
                "line": 1
            }
        }`, TempDirUri())}, session.SessionNotInitialized.Err())
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
            "text": "provider \"test\" {\n\n}\n",
            "uri": "%s/main.tf"
        }
    }`, TempDirUri())})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
            "textDocument": {
                "uri": "%s/main.tf"
            },
            "position": {
                "character": 0,
                "line": 1
            }
        }`, TempDirUri())}, `{
            "jsonrpc": "2.0",
            "id": 3,
            "result": {
                "isIncomplete": false,
                "items": [
                    {
                        "label":"anonymous",
                        "kind":5,
                        "detail":"Optional, number",
                        "documentation":"Desc 1",
                        "insertTextFormat":1,
                        "textEdit": {
                            "range": {
                                "start": {
                                    "line": 1, 
                                    "character": 0
                                },
                                "end": {
                                    "line": 1, 
                                    "character": 0
                                }
                            },
                            "newText": "anonymous"
                        }
                    },
                    {
                        "label":"base_url",
                        "kind":5,
                        "detail":"Optional, string",
                        "documentation":"Desc 2",
                        "insertTextFormat":1,
                        "textEdit": {
                            "range": {
                                "start": {
                                    "line": 1, 
                                    "character": 0
                                },
                                "end": {
                                    "line": 1, 
                                    "character": 0
                                }
                            },
                            "newText": "base_url"
                        }
                    },
                    {
                        "label":"individual",
                        "kind":5,
                        "detail":"Optional, bool",
                        "documentation":"Desc 3",
                        "insertTextFormat":1,
                        "textEdit": {
                            "range": {
                                "start": {
                                    "line": 1,
                                    "character": 0
                                },
                                "end": {
                                    "line": 1,
                                    "character": 0
                                }
                            },
                            "newText": "individual"
                        }
                    }
                ]
            }
        }`)
}

const testSchemaOutput = `{
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
              "description_kind": "plaintext",
              "optional": true
            },
            "base_url": {
              "type": "string",
              "description": "Desc **2**",
              "description_kind": "markdown",
              "optional": true
            },
            "individual": {
              "type": "bool",
              "description": "Desc _3_",
              "description_kind": "markdown",
              "optional": true
            }
          },
          "block_types": {
            "single_block": {
              "nesting_mode": "single",
              "block": {
                "description": "Single Block",
                "description_kind": "markdown",
                "attributes": {
                  "nested_string": {
                    "type": "string",
                    "optional": true
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "resource_schemas": {
    "test_resource_1": {
      "version": 0,
      "block": {
        "description": "Resource 1 description",
        "description_kind": "markdown",
        "attributes": {
          "deprecated_attr": {
            "deprecated": true
          }
        }
      }
    }
  }
}`
