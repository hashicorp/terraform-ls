package handlers

import (
	"testing"

	"github.com/creachadair/jrpc2/code"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/errors"
)

func TestLangServer_completionWithoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: `{
			"textDocument": {
				"uri": "file:///var/main.tf"
			},
			"position": {
				"character": 0,
				"line": 1
			}
		}`}, errors.ServerNotInitialized.Err())
}

func TestLangServer_completeWithIncompatibleTerraformVersion(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(&exec.Mock{
		Args:   []string{"version"},
		Stdout: "Terraform v0.11.0\n",
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: `{
	    "capabilities": {},
	    "processId": 12345
	}`})
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: `{}`})
	ls.Notify(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: `{
		"textDocument": {
			"uri": "file:///tmp/test.tf",
			"text": "provider \"github\" {\n\n}\n",
			"languageId": "terraform",
			"version": 0
		}
	}`})

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: `{
			"textDocument": {
				"uri": "file:///tmp/test.tf"
			},
			"position": {
				"character": 0,
				"line": 1
			}
		}`}, code.SystemError.Err())
}
