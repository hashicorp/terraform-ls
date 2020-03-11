package handlers

import (
	"testing"

	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/errors"
)

func TestLangServer_didOpenWithoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: `{
		"textDocument": {
			"version": 0,
			"languageId": "terraform",
			"text": "provider \"github\" {\n\n}\n",
			"uri": "file:///var/main.tf"
		}
	}`}, errors.ServerNotInitialized.Err())
}
