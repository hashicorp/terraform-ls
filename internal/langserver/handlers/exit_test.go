// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"testing"
	"time"

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

	time.Sleep(10 * time.Millisecond)

	if !ms.StopFuncCalled() {
		t.Fatal("Expected service stop function to be called")
	}

	if ls.StopFuncCalled() {
		t.Fatal("Expected server stop function not to be called")
	}
}
