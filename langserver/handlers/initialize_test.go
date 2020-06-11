package handlers

import (
	"fmt"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/langserver"
)

func TestInitialize_twice(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(map[string]*rootmodule.RootModuleMock{
		TempDir().Dir(): {TerraformExecQueue: validTfMockCalls()},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDir().URI())})
	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDir().URI())}, code.SystemError.Err())
}

func TestInitialize_withIncompatibleTerraformVersion(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(map[string]*rootmodule.RootModuleMock{
		TempDir().Dir(): {
			TerraformExecQueue: &exec.MockCall{
				Args:   []string{"version"},
				Stdout: "Terraform v0.11.0\n",
			},
		},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "processId": 12345,
	    "rootUri": %q
	}`, TempDir().URI())}, code.SystemError.Err())
}

func TestInitialize_withInvalidRootURI(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(map[string]*rootmodule.RootModuleMock{
		TempDir().Dir(): {TerraformExecQueue: validTfMockCalls()},
	}))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: `{
	    "capabilities": {},
	    "processId": 12345,
	    "rootUri": "meh"
	}`}, code.SystemError.Err())
}
