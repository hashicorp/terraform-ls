package handlers

import (
	"errors"
	"fmt"
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/session"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_formattingWithoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"github\" {}",
			"uri": "%s/main.tf"
		}
	}`, TempDir(t).URI())}, session.SessionNotInitialized.Err())
}

func TestLangServer_formatting_basic(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TfExecutorFactory: exec.NewMockExecutor([]*mock.Call{
			{
				Method:        "Version",
				Repeatability: 1,
				Arguments: []interface{}{
					mock.AnythingOfType(""),
				},
				ReturnArguments: []interface{}{
					version.Must(version.NewVersion("0.12.0")),
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
				Method:        "Format",
				Repeatability: 1,
				Arguments: []interface{}{
					mock.AnythingOfType(""),
					[]byte("provider  \"test\"   {\n\n}\n"),
				},
				ReturnArguments: []interface{}{
					[]byte("provider \"test\" {\n\n}\n"),
					nil,
				},
			},
		}),
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDir(t).URI())})
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
			"text": "provider  \"test\"   {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, TempDir(t).URI())})
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			}
		}`, TempDir(t).URI())}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": [
				{
					"range": {
						"start": { "line": 0, "character": 0 },
						"end": { "line": 1, "character": 0 }
					},
					"newText": "provider \"test\" {\n"
				}
			]
		}`)
}

func TestLangServer_formatting_oldVersion(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TfExecutorFactory: exec.NewMockExecutor([]*mock.Call{
			{
				Method:        "Version",
				Repeatability: 1,
				Arguments: []interface{}{
					mock.AnythingOfType(""),
				},
				ReturnArguments: []interface{}{
					version.Must(version.NewVersion("0.7.6")),
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
				Method:        "Format",
				Repeatability: 1,
				Arguments: []interface{}{
					mock.AnythingOfType(""),
					[]byte("provider  \"test\"   {\n\n}\n"),
				},
				ReturnArguments: []interface{}{
					nil,
					errors.New("not implemented"),
				},
			},
		}),
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
	    "capabilities": {},
	    "rootUri": %q,
	    "processId": 12345
	}`, TempDir(t).URI())})
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
			"text": "provider  \"test\"   {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, TempDir(t).URI())})
	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/formatting",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			}
		}`, TempDir(t).URI())}, code.SystemError.Err())
}
