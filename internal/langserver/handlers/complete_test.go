// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	osExec "os/exec"
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
	"github.com/hashicorp/terraform-ls/internal/walker"
	"github.com/stretchr/testify/mock"
)

func requireCompletionHasLabels(t *testing.T, result json.RawMessage, want ...string) {
	t.Helper()

	type completionItem struct {
		Label string `json:"label"`
	}
	type completionList struct {
		IsIncomplete bool             `json:"isIncomplete"`
		Items        []completionItem `json:"items"`
	}

	var labels []string
	var cl completionList
	if err := json.Unmarshal(result, &cl); err == nil {
		labels = make([]string, 0, len(cl.Items))
		for _, it := range cl.Items {
			labels = append(labels, it.Label)
		}
	} else {
		// Some servers return an array of CompletionItem instead of CompletionList.
		var items []completionItem
		if err2 := json.Unmarshal(result, &items); err2 != nil {
			t.Fatalf("failed to unmarshal completion result: %v\nraw: %s", err2, string(result))
		}
		labels = make([]string, 0, len(items))
		for _, it := range items {
			labels = append(labels, it.Label)
		}
	}

	labelsMap := make(map[string]struct{}, len(labels))
	for _, l := range labels {
		labelsMap[l] = struct{}{}
	}
	for _, w := range want {
		if _, ok := labelsMap[w]; !ok {
			t.Fatalf("completion missing label %q\nlabels: %v", w, labels)
		}
	}
}

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

func TestModuleCompletion_withValidData_basic(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte("provider \"test\" {\n\n}\n"), 0o755)
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

	wc := walker.NewWalkerCollector()

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
	waitForAllJobs(t, ss)

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
				"items": [
					{
						"label": "alias",
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

// verify that for old versions we serve earliest available (v0.12) schema
func TestModuleCompletion_withValidData_tooOldVersion(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte("variable \"test\" {\n\n}\n"), 0o755)
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

	wc := walker.NewWalkerCollector()

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
							version.Must(version.NewVersion("0.10.0")),
							nil,
							nil,
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
			"text": "variable \"test\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

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
				"items": [
					{
						"label": "default",
						"kind": 10,
						"detail": "optional, any type",
						"documentation": "Default value to use when variable is not explicitly set",
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
							"newText": "default"
						}
					},
					{
						"label": "description",
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Description to document the purpose of the variable and what value is expected",
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
							"newText": "description"
						}
					},
					{
						"label": "type",
						"kind": 10,
						"detail": "optional, type",
						"documentation": "Type constraint restricting the type of value to accept, e.g. string or list(string)",
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
							"newText": "type"
						}
					}
				]
			}
		}`)
}

// verify that for unknown new versions we serve latest available schema
func TestModuleCompletion_withValidData_tooNewVersion(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte("variable \"test\" {\n\n}\n"), 0o755)
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

	wc := walker.NewWalkerCollector()

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
							version.Must(version.NewVersion("999.999.999")),
							nil,
							nil,
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
			"text": "variable \"test\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

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
				"items": [
					{
						"label": "default",
						"kind": 10,
						"detail": "optional, any type",
						"documentation": "Default value to use when variable is not explicitly set",
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
							"newText": "default"
						}
					},
					{
						"label": "description",
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Description to document the purpose of the variable and what value is expected",
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
							"newText": "description"
						}
					},
					{
						"label": "ephemeral",
						"kind": 10,
						"detail": "optional, bool",
						"documentation": "Whether the value is ephemeral and should not be persisted in the state",
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
							"newText": "ephemeral"
						}
					},
					{
						"label": "nullable",
						"kind": 10,
						"detail": "optional, bool",
						"documentation": "Specifies whether null is a valid value for this variable",
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
							"newText": "nullable"
						}
					},
					{
						"label": "sensitive",
						"kind": 10,
						"detail": "optional, bool",
						"documentation": "Whether the variable contains sensitive material and should be hidden in the UI",
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
							"newText": "sensitive"
						}
					},
					{
						"label": "type",
						"kind": 10,
						"detail": "optional, type",
						"documentation": "Type constraint restricting the type of value to accept, e.g. string or list(string)",
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
							"newText": "type"
						}
					},
					{
						"label": "validation",
						"kind": 7,
						"detail": "Block",
						"documentation": "Custom validation rule to restrict what value is expected for the variable",
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
							"newText": "validation"
						}
					}
				]
			}
		}`)
}

func TestModuleCompletion_withValidDataAndSnippets(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())
	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte("provider \"test\" {\n\n}\n"), 0o755)
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
	wc := walker.NewWalkerCollector()

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
	waitForAllJobs(t, ss)

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
				"items": [
					{
						"label": "alias",
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
								"type": "string",
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
	wc := walker.NewWalkerCollector()

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
	waitForAllJobs(t, ss)

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
				"items": [
					{
						"label": "test",
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

func TestCompletion_withValidData_complexVariableAttributeAccess(t *testing.T) {
	// Verifies completion for deep attribute access chains like:
	// var.infrastructure_config.network_configuration.subnets[0].security.<...>
	// var.infrastructure_config.compute_resources["database"].storage.backup.<...>
	// var.recursive_config.layers["frontend"].sub_layers["ui"].components[0].metadata.<...>
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
	wc := walker.NewWalkerCollector()

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
		}`, tmpDir.URI),
	})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{Method: "initialized", ReqParams: "{}"})

	// Variable definitions (types drive completion)
	ls.Notify(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"version": 0,
				"languageId": "terraform",
				"text": %q,
				"uri": "%s/variables.tf"
			}
		}`, `variable "infrastructure_config" {
  type = object({
    project_metadata = object({
      name        = string
      environment = string
      owner       = string
      tags        = map(string)
    })
    network_configuration = object({
      vnet = object({
        name          = string
        address_space = list(string)
        dns_servers   = list(string)
      })
      subnets = list(object({
        name   = string
        prefix = string
        security = object({
          nsg_id            = string
          allow_internet    = bool
          flow_logs_enabled = bool
        })
      }))
    })
    compute_resources = map(object({
      vm_size = string
      storage = object({
        disk_size_gb = number
        tier         = string
        backup = object({
          enabled        = bool
          retention_days = number
        })
      })
      monitoring = object({
        enabled = bool
        alerts  = list(string)
      })
    }))
  })
}
variable "recursive_config" {
  type = object({
    layers = map(object({
      sub_layers = map(object({
        components = list(object({
          id    = string
          props = map(string)
          metadata = object({
            created_by = string
            version    = number
          })
        }))
      }))
    }))
  })
}
`, tmpDir.URI),
	})

	// Open locals file and check completions at various points.
	ls.Notify(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"version": 0,
				"languageId": "terraform",
				"text": %q,
				"uri": "%s/locals.tf"
			}
		}`, `locals {
  env_name = var.infrastructure_config.
}
`, tmpDir.URI),
	})
	waitForAllJobs(t, ss)

	// 1) Top-level attributes of infrastructure_config
	rsp := ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {"uri": "%s/locals.tf"},
			"position": {"line": 1, "character": 39}
		}`, tmpDir.URI),
	})
	requireCompletionHasLabels(t, rsp.Result,
		"var.infrastructure_config.project_metadata",
		"var.infrastructure_config.network_configuration",
		"var.infrastructure_config.compute_resources",
	)

	// 2) project_metadata.<...>
	ls.Notify(t, &langserver.CallRequest{
		Method: "textDocument/didChange",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {"version": 1, "uri": "%s/locals.tf"},
			"contentChanges": [{"text": %q}]
		}`, tmpDir.URI, `locals {
  env_name = var.infrastructure_config.project_metadata.
}
`),
	})
	waitForAllJobs(t, ss)

	rsp = ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {"uri": "%s/locals.tf"},
			"position": {"line": 1, "character": 56}
		}`, tmpDir.URI),
	})
	requireCompletionHasLabels(t, rsp.Result,
		"var.infrastructure_config.project_metadata.environment",
		"var.infrastructure_config.project_metadata.name",
		"var.infrastructure_config.project_metadata.owner",
		"var.infrastructure_config.project_metadata.tags",
	)

	// 3) subnets[0].security.<...>
	ls.Notify(t, &langserver.CallRequest{
		Method: "textDocument/didChange",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {"version": 2, "uri": "%s/locals.tf"},
			"contentChanges": [{"text": %q}]
		}`, tmpDir.URI, `locals {
  first_subnet_security = var.infrastructure_config.network_configuration.subnets[0].security.
}
`),
	})
	waitForAllJobs(t, ss)

	rsp = ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {"uri": "%s/locals.tf"},
			"position": {"line": 1, "character": 94}
		}`, tmpDir.URI),
	})
	requireCompletionHasLabels(t, rsp.Result,
		"var.infrastructure_config.network_configuration.subnets[0].security.allow_internet",
		"var.infrastructure_config.network_configuration.subnets[0].security.flow_logs_enabled",
		"var.infrastructure_config.network_configuration.subnets[0].security.nsg_id",
	)

	// 4) compute_resources["database"].storage.backup.<...>
	ls.Notify(t, &langserver.CallRequest{
		Method: "textDocument/didChange",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {"version": 3, "uri": "%s/locals.tf"},
			"contentChanges": [{"text": %q}]
		}`, tmpDir.URI, `locals {
  db_backup_retention = var.infrastructure_config.compute_resources["database"].storage.backup.
}
`),
	})
	waitForAllJobs(t, ss)

	rsp = ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {"uri": "%s/locals.tf"},
			"position": {"line": 1, "character": 95}
		}`, tmpDir.URI),
	})
	requireCompletionHasLabels(t, rsp.Result,
		"var.infrastructure_config.compute_resources[\"database\"].storage.backup.enabled",
		"var.infrastructure_config.compute_resources[\"database\"].storage.backup.retention_days",
	)

	// 5) recursive_config ... metadata.<...>
	ls.Notify(t, &langserver.CallRequest{
		Method: "textDocument/didChange",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {"version": 4, "uri": "%s/locals.tf"},
			"contentChanges": [{"text": %q}]
		}`, tmpDir.URI, `locals {
  component_version = var.recursive_config.layers["frontend"].sub_layers["ui"].components[0].metadata.
}
`),
	})
	waitForAllJobs(t, ss)

	rsp = ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {"uri": "%s/locals.tf"},
			"position": {"line": 1, "character": 102}
		}`, tmpDir.URI),
	})
	requireCompletionHasLabels(t, rsp.Result,
		"var.recursive_config.layers[\"frontend\"].sub_layers[\"ui\"].components[0].metadata.created_by",
		"var.recursive_config.layers[\"frontend\"].sub_layers[\"ui\"].components[0].metadata.version",
	)
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
	// Keep a valid config for `terraform get`, then overwrite to the
	// completion test content (which may be syntactically incomplete).
	mainCfg := `module "refname" {
  source = "./submodule"

}

output "test" {
	value = 42
}
`
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "main.tf"), mainCfg)

	tfExec := tfExecutor(t, tmpDir.Path(), "1.0.2")
	err := tfExec.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	mainCfg = `module "refname" {
  source = "./submodule"

}

	output "test" {
	  value = module.refname.
	}
`
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "main.tf"), mainCfg)

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()
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
	waitForAllJobs(t, ss)

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
				"items": [
					{
						"label": "providers",
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

	rsp := ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 25,
				"line": 6
			}
		}`, tmpDir.URI),
	})
	requireCompletionHasLabels(t, rsp.Result, "module.refname")
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
	// Keep a valid config for `terraform get`, then overwrite to the
	// completion test content (which may be syntactically incomplete).
	mainCfg := `module "alpha" {
  source = "./submodule-alpha"

}
module "beta" {
  source = "./submodule-beta"

}

output "test" {
	value = 42
}
`
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "main.tf"), mainCfg)

	tfExec := tfExecutor(t, tmpDir.Path(), "1.0.2")
	err := tfExec.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}

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
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "main.tf"), mainCfg)

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

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
	waitForAllJobs(t, ss)

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
				"items": [
					{
						"label": "alpha-var",
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
				"items": [
					{
						"label": "beta-var",
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
				"items": [
					{
						"label": "module.alpha",
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
	wc := walker.NewWalkerCollector()

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
	waitForAllJobs(t, ss)

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
				"items": [
					{
						"label": "var.aaa",
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
	// Prefer using Terraform already on PATH.
	// CI and local dev envs often preinstall Terraform, while hc-install downloads
	// can fail due to network/proxy restrictions.
	if p, err := osExec.LookPath("terraform"); err == nil {
		tfExec, err := exec.NewExecutor(workdir, p)
		if err != nil {
			t.Fatal(err)
		}
		return tfExec
	}

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
