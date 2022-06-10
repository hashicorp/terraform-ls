package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	hcinstall "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
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
		}`, TempDir(t).URI)}, session.SessionNotInitialized.Err())
}

func TestModuleCompletion_withValidData(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	var testSchema tfjson.ProviderSchemas
	err := json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	wc := module.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		StateStore: ss,
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
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
		},
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
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"test\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})

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
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"itemDefaults": {
					"editRange": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 0}
					}
				},
				"items": [
					{
						"label": "alias",
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, string",
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
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, number",
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
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, string",
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
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, bool",
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
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, string",
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
	InitPluginCache(t, tmpDir.Path())

	var testSchema tfjson.ProviderSchemas
	err := json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := module.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
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
		},
		StateStore:      ss,
		WalkerCollector: wc,
	}))
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
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
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
	}`, tmpDir.URI)})

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
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"itemDefaults": {
					"editRange": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 0}
					}
				},
				"items": [
					{
						"label": "alias",
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, string",
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
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, number",
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
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, string",
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
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, bool",
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
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, string",
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
				},
				"test_resource_2": {
					"version": 0,
					"block": {
						"description_kind": "markdown",
						"attributes": {
							"optional_attr": {
								"type": "string",
								"description_kind": "plain",
								"optional": true
							}
						},
						"block_types": {
							"setting": {
								"nesting_mode": "set",
								"block": {
									"attributes": {
										"name": {
											"type": "string",
											"description_kind": "plain",
											"required": true
										},
										"value": {
											"type": "string",
											"description_kind": "plain",
											"required": true
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}`

func TestVarsCompletion_withValidData(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	var testSchema tfjson.ProviderSchemas
	err := json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := module.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
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
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "variable \"test\" {\n type=string\n}\n",
			"uri": "%s/variables.tf"
		}
	}`, tmpDir.URI)})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform-vars",
			"uri": "%s/terraform.tfvars"
		}
	}`, tmpDir.URI)})

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
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 4,
			"result": {
				"isIncomplete": false,
				"itemDefaults": {
					"editRange": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 0}
					}
				},
				"items": [
					{
						"label": "test",
						"labelDetails": {},
						"kind": 10,
						"detail": "required, string",
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

func TestCompletion_moduleWithValidData(t *testing.T) {
	tmpDir := TempDir(t)

	writeContentToFile(t, filepath.Join(tmpDir.Path(), "submodule", "main.tf"), `variable "testvar" {
	type = string
}

output "testout" {
	value = 42
}
`)
	mainCfg := `module "refname" {
  source = "./submodule"

}

output "test" {

}
`
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "main.tf"), mainCfg)
	mainCfg = `module "refname" {
  source = "./submodule"

}

output "test" {
  value = module.refname.
}
`

	tfExec := tfExecutor(t, tmpDir.Path(), "1.0.2")
	err := tfExec.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := module.NewWalkerCollector()
	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
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
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": %q,
			"uri": "%s/main.tf"
		}
	}`, mainCfg, tmpDir.URI)})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 2
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"itemDefaults": {
					"editRange": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 0}
					}
				},
				"items": [
					{
						"label": "providers",
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, map of provider references",
						"documentation": "Explicit mapping of providers which the module uses",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 2,
									"character": 0
								},
								"end": {
									"line": 2,
									"character": 0
								}
							},
							"newText": "providers"
						}
					},
					{
						"label": "testvar",
						"labelDetails": {},
						"kind": 10,
						"detail": "required, string",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 2,
									"character": 0
								},
								"end": {
									"line": 2,
									"character": 0
								}
							},
							"newText": "testvar"
						}
					},
					{
						"label": "version",
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Constraint to set the version of the module, e.g. ~\u003e 1.0. Only applicable to modules in a module registry.",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 2,
									"character": 0
								},
								"end": {
									"line": 2,
									"character": 0
								}
							},
							"newText": "version"
						}
					}
				]
			}
		}`)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 25,
				"line": 6
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 4,
			"result": {
				"isIncomplete": false,
				"itemDefaults": {
					"editRange": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 0}
					}
				},
				"items": [
					{
						"label": "module.refname.testout",
						"labelDetails": {},
						"kind": 6,
						"detail": "number",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 6,
									"character": 10
								},
								"end": {
									"line": 6,
									"character": 25
								}
							},
							"newText": "module.refname.testout"
						}
					}
				]
			}
		}`)
}

func TestCompletion_multipleModulesWithValidData(t *testing.T) {
	tmpDir := TempDir(t)

	writeContentToFile(t, filepath.Join(tmpDir.Path(), "submodule-alpha", "main.tf"), `
variable "alpha-var" {
	type = string
}

output "alpha-out" {
	value = 1
}
`)
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "submodule-beta", "main.tf"), `
variable "beta-var" {
	type = number
}

output "beta-out" {
	value = 2
}
`)
	mainCfg := `module "alpha" {
  source = "./submodule-alpha"

}
module "beta" {
  source = "./submodule-beta"

}

output "test" {

}
`
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "main.tf"), mainCfg)
	mainCfg = `module "alpha" {
  source = "./submodule-alpha"

}
module "beta" {
  source = "./submodule-beta"

}

output "test" {
  value = module.
}
`

	tfExec := tfExecutor(t, tmpDir.Path(), "1.0.2")
	err := tfExec.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := module.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
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
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": %q,
			"uri": "%s/main.tf"
		}
	}`, mainCfg, tmpDir.URI)})

	// first module
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 2
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"itemDefaults": {
					"editRange": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 0}
					}
				},
				"items": [
					{
						"label": "alpha-var",
						"labelDetails": {},
						"kind": 10,
						"detail": "required, string",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 2,
									"character": 0
								},
								"end": {
									"line": 2,
									"character": 0
								}
							},
							"newText": "alpha-var"
						}
					},
					{
						"label": "providers",
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, map of provider references",
						"documentation": "Explicit mapping of providers which the module uses",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 2,
									"character": 0
								},
								"end": {
									"line": 2,
									"character": 0
								}
							},
							"newText": "providers"
						}
					},
					{
						"label": "version",
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Constraint to set the version of the module, e.g. ~\u003e 1.0. Only applicable to modules in a module registry.",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 2,
									"character": 0
								},
								"end": {
									"line": 2,
									"character": 0
								}
							},
							"newText": "version"
						}
					}
				]
			}
		}`)
	// second module
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 6
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 4,
			"result": {
				"isIncomplete": false,
				"itemDefaults": {
					"editRange": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 0}
					}
				},
				"items": [
					{
						"label": "beta-var",
						"labelDetails": {},
						"kind": 10,
						"detail": "required, number",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 6,
									"character": 0
								},
								"end": {
									"line": 6,
									"character": 0
								}
							},
							"newText": "beta-var"
						}
					},
					{
						"label": "providers",
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, map of provider references",
						"documentation": "Explicit mapping of providers which the module uses",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 6,
									"character": 0
								},
								"end": {
									"line": 6,
									"character": 0
								}
							},
							"newText": "providers"
						}
					},
					{
						"label": "version",
						"labelDetails": {},
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Constraint to set the version of the module, e.g. ~\u003e 1.0. Only applicable to modules in a module registry.",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 6,
									"character": 0
								},
								"end": {
									"line": 6,
									"character": 0
								}
							},
							"newText": "version"
						}
					}
				]
			}
		}`)
	// outputs
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 17,
				"line": 10
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 5,
			"result": {
				"isIncomplete": false,
				"itemDefaults": {
					"editRange": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 0}
					}
				},
				"items": [
					{
						"label": "module.alpha",
						"labelDetails": {},
						"kind": 6,
						"detail": "object",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 10,
									"character": 10
								},
								"end": {
									"line": 10,
									"character": 17
								}
							},
							"newText": "module.alpha"
						}
					},
					{
						"label": "module.beta",
						"labelDetails": {},
						"kind": 6,
						"detail": "object",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 10,
									"character": 10
								},
								"end": {
									"line": 10,
									"character": 17
								}
							},
							"newText": "module.beta"
						}
					}
				]
			}
		}`)
}

func TestVarReferenceCompletion_withValidData(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	variableDecls := `variable "aaa" {}
variable "bbb" {}
variable "ccc" {}
`
	f, err := os.Create(filepath.Join(tmpDir.Path(), "variables.tf"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(variableDecls)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := module.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
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
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "output \"test\" {\n  value = var.\n}\n",
			"uri": "%s/outputs.tf"
		}
	}`, tmpDir.URI)})
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/outputs.tf"
			},
			"position": {
				"character": 14,
				"line": 1
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"itemDefaults": {
					"editRange": {
						"start": {"line": 0, "character": 0},
						"end": {"line": 0, "character": 0}
					}
				},
				"items": [
					{
						"label": "var.aaa",
						"labelDetails": {},
						"kind": 6,
						"detail": "dynamic",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 10
								},
								"end": {
									"line": 1,
									"character": 14
								}
							},
							"newText": "var.aaa"
						}
					},
					{
						"label": "var.bbb",
						"labelDetails": {},
						"kind": 6,
						"detail": "dynamic",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 10
								},
								"end": {
									"line": 1,
									"character": 14
								}
							},
							"newText": "var.bbb"
						}
					},
					{
						"label": "var.ccc",
						"labelDetails": {},
						"kind": 6,
						"detail": "dynamic",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 10
								},
								"end": {
									"line": 1,
									"character": 14
								}
							},
							"newText": "var.ccc"
						}
					}
				]
			}
		}`)
}

func tfExecutor(t *testing.T, workdir, tfVersion string) exec.TerraformExecutor {
	ctx := context.Background()
	installDir := filepath.Join(t.TempDir(), "hcinstall")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(installDir); err != nil {
			t.Fatal(err)
		}
	})

	i := hcinstall.NewInstaller()
	v := version.Must(version.NewVersion(tfVersion))

	execPath, err := i.Ensure(ctx, []src.Source{
		&fs.ExactVersion{
			Product: product.Terraform,
			Version: v,
		},
		&releases.ExactVersion{
			Product:    product.Terraform,
			Version:    v,
			InstallDir: installDir,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := i.Remove(ctx); err != nil {
			t.Fatal(err)
		}
	})

	tfExec, err := exec.NewExecutor(workdir, execPath)
	if err != nil {
		t.Fatal(err)
	}
	return tfExec
}

func writeContentToFile(t *testing.T, path string, content string) {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
}
