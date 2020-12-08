package handlers

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
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
			"uri": %q
		}
	}`, testFileURI)})

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q
	}`, cmd.Name("rootmodules"))}, code.InvalidParams.Err())
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
			"uri": %q
		}
	}`, testFileURI)})

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: fmt.Sprintf(`{
		"command": %q,
		"arguments": ["uri=%s"] 
	}`, cmd.Name("rootmodules"), testFileURI)}, fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 3,
		"result": {
			"responseVersion": 0,
			"doneLoading": true,
			"rootModules": [
				{
					"uri": %q,
					"name": %q
				}
			]
		}
	}`, tmpDir.URI(), t.Name()))
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
		"processId": 12345
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
		"command": %q,
		"arguments": ["uri=%s"] 
	}`, cmd.Name("rootmodules"), module.URI())}, fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 2,
		"result": {
			"responseVersion": 0,
			"doneLoading": true,
			"rootModules": [
				{
					"uri": %q,
					"name": %q
				},
				{
					"uri": %q,
					"name": %q
				},
				{
					"uri": %q,
					"name": %q
				}
			]
		}
	}`, dev.URI(), filepath.Join("env", "dev"), prod.URI(), filepath.Join("env", "prod"), staging.URI(), filepath.Join("env", "staging")))
}
