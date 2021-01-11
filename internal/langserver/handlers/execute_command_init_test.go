package handlers

import (
	"errors"
	"fmt"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_workspaceExecuteCommand_init_argumentError(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.URI())

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		Modules: map[string]*module.ModuleMock{
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
	}`, cmd.Name("terraform.init"))}, code.InvalidParams.Err())
}

func TestLangServer_workspaceExecuteCommand_init_basic(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.URI())

	tfMockCalls := exec.NewMockExecutor([]*mock.Call{
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
			Method:        "Init",
			Repeatability: 1,
			Arguments: []interface{}{
				mock.AnythingOfType(""),
			},
			ReturnArguments: []interface{}{
				nil,
			},
		},
	})

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		Modules: map[string]*module.ModuleMock{
			tmpDir.Dir(): {
				TfExecFactory: tfMockCalls,
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
	}`, cmd.Name("terraform.init"), tmpDir.URI())}, `{
		"jsonrpc": "2.0",
		"id": 3,
		"result": null
	}`)
}

func TestLangServer_workspaceExecuteCommand_init_error(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.URI())

	tfMockCalls := exec.NewMockExecutor([]*mock.Call{
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
			Method:        "Init",
			Repeatability: 1,
			Arguments: []interface{}{
				mock.AnythingOfType(""),
			},
			ReturnArguments: []interface{}{
				errors.New("something bad happened"),
			},
		},
	})

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		Modules: map[string]*module.ModuleMock{
			tmpDir.Dir(): {
				TfExecFactory: tfMockCalls,
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
		"command": %q,
		"arguments": ["uri=%s"]
	}`, cmd.Name("terraform.init"), testFileURI)}, code.SystemError.Err())
}
