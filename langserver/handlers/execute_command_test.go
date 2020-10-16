package handlers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/langserver"
)

func TestLangServer_workspaceExecuteCommand_noCommandHandlerError(t *testing.T) {
	tmpDir := TempDir(t)
	testFileURI := fmt.Sprintf("%s/main.tf", tmpDir.Dir())
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
			"text": "provider \"github\"\n\n}\n",
			"uri": "%s"
		}
	}`, testFileURI)})

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "workspace/executeCommand",
		ReqParams: `{
		"command": "notfound"
	}`}, commandHandlerNotFound.Err())
}
