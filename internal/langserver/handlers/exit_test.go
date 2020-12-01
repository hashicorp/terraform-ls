package handlers

import (
	"testing"

	"github.com/hashicorp/terraform-ls/internal/langserver"
)

func TestExit(t *testing.T) {
	ms := newMockSession(nil)
	ls := langserver.NewLangServerMock(t, ms.new)
	stop := ls.Start(t)
	defer stop()

	ls.Notify(t, &langserver.CallRequest{
		Method:    "exit",
		ReqParams: `{}`})

	if !ms.StopFuncCalled() {
		t.Fatal("Expected service stop function to be called")
	}

	if ls.StopFuncCalled() {
		t.Fatal("Expected server stop function not to be called")
	}
}
