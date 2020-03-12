package handlers

import (
	"testing"

	"github.com/hashicorp/terraform-ls/langserver"
)

func TestExit(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMock(nil))
	stop := ls.Start(t)
	defer stop()

	ls.Notify(t, &langserver.CallRequest{
		Method:    "exit",
		ReqParams: `{}`})

	if !ls.StopFuncCalled() {
		t.Fatal("Expected stop function to be called")
	}
}
