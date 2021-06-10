package handlers

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/stretchr/testify/mock"
)

func TestModuleCompletion_withoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(nil))
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
		}`, TempDir(t).URI())}, session.SessionNotInitialized.Err())
}

func TestModuleCompletion_withValidData(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Dir())

	var testSchema tfjson.ProviderSchemas
	err := json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Dir(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("0.12.0")),
							nil,
							nil,
						},
					},
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
						},
					},
					{
						Method:        "ProviderSchemas",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							&testSchema,
							nil,
						},
					},
				},
			},
		}}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
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
			"text": "provider \"test\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI())})

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
		}`, tmpDir.URI())}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "alias",
						"kind": 10,
						"detail": "Optional, string",
						"documentation": "Alias for using the same provider with different configurations for different resources, e.g. eu-west",
						"insertTextFormat": 1,
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
							"newText": "alias"
						}
					},
					{
						"label": "anonymous",
						"kind": 10,
						"detail": "Optional, number",
						"documentation": "Desc 1",
						"insertTextFormat": 1,
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
						"label": "base_url",
						"kind": 10,
						"detail": "Optional, string",
						"documentation": "Desc 2",
						"insertTextFormat": 1,
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
						"label": "individual",
						"kind": 10,
						"detail": "Optional, bool",
						"documentation": "Desc 3",
						"insertTextFormat": 1,
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
					},
					{
						"label": "version",
						"kind": 10,
						"detail": "Optional, string",
						"documentation": "Specifies a version constraint for the provider, e.g. ~\u003e 1.0",
						"insertTextFormat": 1,
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
							"newText": "version"
						}
					}
				]
			}
		}`)
}

func TestModuleCompletion_withValidDataAndSnippets(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Dir())

	var testSchema tfjson.ProviderSchemas
	err := json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Dir(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("0.12.0")),
							nil,
							nil,
						},
					},
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
						},
					},
					{
						Method:        "ProviderSchemas",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							&testSchema,
							nil,
						},
					},
				},
			},
		}}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {
			"textDocument": {
        "completion": {
          "completionItem": {
            "snippetSupport": true
          }
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
			"text": "provider \"test\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI())})

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
		}`, tmpDir.URI())}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "alias",
						"kind": 10,
						"detail": "Optional, string",
						"documentation": "Alias for using the same provider with different configurations for different resources, e.g. eu-west",
						"insertTextFormat": 2,
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
							"newText": "alias = \"${1:value}\""
						}
					},
					{
						"label": "anonymous",
						"kind": 10,
						"detail": "Optional, number",
						"documentation": "Desc 1",
						"insertTextFormat": 2,
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
							"newText": "anonymous = "
						},
						"command": {
							"title": "Suggest",
							"command": "editor.action.triggerSuggest"
						}
					},
					{
						"label": "base_url",
						"kind": 10,
						"detail": "Optional, string",
						"documentation": "Desc 2",
						"insertTextFormat": 2,
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
							"newText": "base_url = "
						},
						"command": {
							"title": "Suggest",
							"command": "editor.action.triggerSuggest"
						}
					},
					{
						"label": "individual",
						"kind": 10,
						"detail": "Optional, bool",
						"documentation": "Desc 3",
						"insertTextFormat": 2,
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
							"newText": "individual = "
						},
						"command": {
							"title": "Suggest",
							"command": "editor.action.triggerSuggest"
						}
					},
					{
						"label": "version",
						"kind": 10,
						"detail": "Optional, string",
						"documentation": "Specifies a version constraint for the provider, e.g. ~\u003e 1.0",
						"insertTextFormat": 2,
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
							"newText": "version = \"${1:value}\""
						}
					}
				]
			}
		}`)
}

var testModuleSchemaOutput = `{
  "format_version": "0.1",
  "provider_schemas": {
	"test/test": {
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

func TestVarsCompletion_withValidData(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Dir())

	var testSchema tfjson.ProviderSchemas
	err := json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Dir(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("0.12.0")),
							nil,
							nil,
						},
					},
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
						},
					},
					{
						Method:        "ProviderSchemas",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							&testSchema,
							nil,
						},
					},
				},
			},
		}}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
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
			"text": "variable \"test\" {\n type=string\n}\n",
			"uri": "%s/variables.tf"
		}
	}`, tmpDir.URI())})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform-vars",
			"uri": "%s/terraform.tfvars"
		}
	}`, tmpDir.URI())})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/terraform.tfvars"
			},
			"position": {
				"character": 0,
				"line": 0
			}
		}`, tmpDir.URI())}, `{
			"jsonrpc": "2.0",
			"id": 4,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "test",
						"kind": 10,
						"detail": "string",
						"insertTextFormat":1,
						"textEdit": {
							"range": {"start":{"line":0,"character":0}, "end":{"line":0,"character":0}}, 
							"newText":"test"
						}
					}
				]
			}
		}`)
}
