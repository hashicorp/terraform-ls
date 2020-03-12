package handlers

import (
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/langserver"
)

func TestShutdown_twice(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(validTfMockCalls()))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: `{
	    "capabilities": {},
	    "rootUri": "file:///tmp",
	    "processId": 12345
	}`})
	ls.Call(t, &langserver.CallRequest{
		Method: "shutdown", ReqParams: `{}`})

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "shutdown", ReqParams: `{}`},
		code.InvalidRequest.Err())
}
