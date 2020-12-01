package handlers

import (
	"fmt"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
)

func TestShutdown_twice(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		RootModules: map[string]*rootmodule.RootModuleMock{
			TempDir(t).Dir(): {TfExecFactory: validTfMockCalls()},
		}}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDir(t).URI())})
	ls.Call(t, &langserver.CallRequest{
		Method: "shutdown", ReqParams: `{}`})

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "shutdown", ReqParams: `{}`},
		code.InvalidRequest.Err())
}
