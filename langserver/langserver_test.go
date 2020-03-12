package langserver

import (
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/langserver/handlers"
)

func TestLangServer_initalizeAndShutdown(t *testing.T) {
	ls := NewLangServerMock(t, handlers.NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectResponse(t, &CallRequest{
		"initialize",
		`{
	    "capabilities": {
	        "workspace": {},
	        "textDocument": {
	            "synchronization": {
	                "didSave": true,
	                "willSaveWaitUntil": true,
	                "willSave": true
	            }
	        }
	    },
	    "rootPath": "",
	    "rootUri": "",
	    "processId": 12345
	}`}, `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"capabilities": {
				"textDocumentSync": {
					"openClose": true,
					"change": 1
				},
				"completionProvider": {}
			}
		}
	}`)
	ls.CallAndExpectResponse(t, &CallRequest{"shutdown", `{}`},
		`{
		"jsonrpc": "2.0",
		"id": 2,
		"result": null
	}`)

}

func TestLangServer_initializeTwice(t *testing.T) {
	ls := NewLangServerMock(t, handlers.NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &CallRequest{
		"initialize",
		`{
	    "capabilities": {},
	    "processId": 12345
	}`})
	ls.CallAndExpectError(t, &CallRequest{
		"initialize",
		`{
	    "capabilities": {},
	    "processId": 12345
	}`}, code.SystemError.Err())
}

func TestLangServer_shutdownTwice(t *testing.T) {
	ls := NewLangServerMock(t, handlers.NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &CallRequest{
		"initialize",
		`{
	    "capabilities": {},
	    "processId": 12345
	}`})
	ls.Call(t, &CallRequest{"shutdown", `{}`})

	ls.CallAndExpectError(t, &CallRequest{"shutdown", `{}`},
		code.InvalidRequest.Err())
}

func TestLangServer_exit(t *testing.T) {
	ls := NewLangServerMock(t, handlers.NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.Notify(t, &CallRequest{
		"exit",
		`{}`})

	if !ls.StopFuncCalled() {
		t.Fatal("Expected stop function to be called")
	}
}
