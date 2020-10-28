package handlers

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/langserver"
)

func TestLangServer_workspaceExecuteCommand_rootmodules_argumentError(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.URI())
	InitPluginCache(t, tmpDir.Dir())

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		RootModules: map[string]*rootmodule.RootModuleMock{
			tmpDir.Dir(): {
				TfExecFactory: validTfMockCalls(),
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
		"processId": 12345,
		"initializationOptions": {
			"id": "1"
		}
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
			"uri": %q
		}
	}`, testFileURI)})

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": "terraform-ls.rootmodules.1"
	}`)}, code.InvalidParams.Err())
}

func TestLangServer_workspaceExecuteCommand_rootmodules_basic(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.URI())
	InitPluginCache(t, tmpDir.Dir())

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		RootModules: map[string]*rootmodule.RootModuleMock{
			tmpDir.Dir(): {
				TfExecFactory: validTfMockCalls(),
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
		"processId": 12345,
		"initializationOptions": {
			"id": "1"
		}
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
			"uri": %q
		}
	}`, testFileURI)})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": "terraform-ls.rootmodules.1",
		"arguments": ["uri=%s"] 
	}`, testFileURI)}, fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 3,
		"result": {
			"responseVersion": 0,
			"doneLoading": true,
			"rootModules": [
				{
					"uri": %q
				}
			]
		}
	}`, tmpDir.URI()))
}

func TestLangServer_workspaceExecuteCommand_rootmodules_multiple(t *testing.T) {
	testData, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	root := lsp.FileHandlerFromDirPath(filepath.Join(testData, "main-module-multienv"))
	module := lsp.FileHandlerFromDirPath(filepath.Join(testData, "main-module-multienv", "main", "main.tf"))

	dev := lsp.FileHandlerFromDirPath(filepath.Join(testData, "main-module-multienv", "env", "dev"))
	staging := lsp.FileHandlerFromDirPath(filepath.Join(testData, "main-module-multienv", "env", "staging"))
	prod := lsp.FileHandlerFromDirPath(filepath.Join(testData, "main-module-multienv", "env", "prod"))

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		RootModules: map[string]*rootmodule.RootModuleMock{
			dev.Dir(): {
				TfExecFactory: validTfMockCalls(),
			},
			staging.Dir(): {
				TfExecFactory: validTfMockCalls(),
			},
			prod.Dir(): {
				TfExecFactory: validTfMockCalls(),
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
		"processId": 12345,
		"initializationOptions": {
			"id": "1"
		}
	}`, root.URI())})
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})

	// expect module definition to be associated to three rootmodules
	// expect modules to be alphabetically sorted on uri
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": "terraform-ls.rootmodules.1",
		"arguments": ["uri=%s"] 
	}`, module.URI())}, fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 2,
		"result": {
			"responseVersion": 0,
			"doneLoading": true,
			"rootModules": [
				{
					"uri": %q
				},
				{
					"uri": %q
				},
				{
					"uri": %q
				}
			]
		}
	}`, dev.URI(), prod.URI(), staging.URI()))
}
