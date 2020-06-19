package handlers

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/internal/watcher"
	"github.com/hashicorp/terraform-ls/langserver/session"
)

type mockSession struct {
	mockRMs map[string]*rootmodule.RootModuleMock

	stopFunc       func()
	stopFuncCalled bool
}

func (ms *mockSession) new(srvCtx context.Context) session.Session {
	sessCtx, stopSession := context.WithCancel(srvCtx)
	ms.stopFunc = stopSession

	logger := testLogger()
	rmmm := rootmodule.NewRootModuleManagerMock(ms.mockRMs)

	svc := &service{
		logger:               logger,
		srvCtx:               srvCtx,
		sessCtx:              sessCtx,
		stopSession:          ms.stop,
		newRootModuleManager: rmmm,
		newWatcher:           watcher.MockWatcher(),
	}

	return svc
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}

func (ms *mockSession) stop() {
	ms.stopFunc()
	ms.stopFuncCalled = true
}

func (ms *mockSession) StopFuncCalled() bool {
	return ms.stopFuncCalled
}

func newMockSession(mockRMs map[string]*rootmodule.RootModuleMock) *mockSession {
	return &mockSession{mockRMs: mockRMs}
}

func NewMockSession(mockRMs map[string]*rootmodule.RootModuleMock) session.SessionFactory {
	return newMockSession(mockRMs).new
}
