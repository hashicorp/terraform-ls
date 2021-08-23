package handlers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/stretchr/testify/mock"
)

func TestLangServer_codeActionWithoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/codeAction",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"github\" {}",
			"uri": "%s/main.tf"
		}
	}`, TempDir(t).URI())}, session.SessionNotInitialized.Err())
}

func TestLangServer_codeAction_basic(t *testing.T) {
	tmpDir := TempDir(t)

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TerraformCalls: &exec.TerraformMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Dir(): {
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
						Method:        "Format",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
							[]byte("provider  \"test\"   {\n\n      }\n"),
						},
						ReturnArguments: []interface{}{
							[]byte("provider \"test\" {\n\n}\n"),
							nil,
						},
					}},
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
			"text": "provider  \"test\"   {\n\n      }\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI())})
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/codeAction",
		ReqParams: fmt.Sprintf(`{
			"textDocument": { "uri": "%s/main.tf" },
			"range": {
				"start": { "line": 0, "character": 0 },
				"end": { "line": 1, "character": 0 }
			},
			"context": { "diagnostics": [], "only": ["source.fixAll"] }
		}`, tmpDir.URI())}, fmt.Sprintf(`{
			"jsonrpc": "2.0",
			"id": 3,
			"result": [
				{
					"title": "Format Document",
					"kind": "source.fixAll",
					"edit":{
						"changes":{
							"%s/main.tf": [
								{
									"range": {
										"start": {
											"line": 0,
											"character": 0
										},
										"end": {
											"line": 1,
											"character": 0
										}
									},
									"newText": "provider \"test\" {\n"
								},
								{
									"range": {
										"start": {
											"line": 2,
											"character": 0
										},
										"end": {
											"line": 3,
											"character": 0
										}
									},
									"newText": "}\n"
								}
							]
						}
					}
				}
			]
		}`, tmpDir.URI()))
}
