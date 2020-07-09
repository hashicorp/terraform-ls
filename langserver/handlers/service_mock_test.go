package handlers

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/rootmodule"
	"github.com/hashicorp/terraform-ls/internal/watcher"
	"github.com/hashicorp/terraform-ls/langserver/session"
)

type MockSessionInput struct {
	RootModules        map[string]*rootmodule.RootModuleMock
	ManagerTfExecQueue exec.MockItemDispenser
}

type mockSession struct {
	mockInput *MockSessionInput

	stopFunc       func()
	stopFuncCalled bool
}

func (ms *mockSession) new(srvCtx context.Context) session.Session {
	sessCtx, stopSession := context.WithCancel(srvCtx)
	ms.stopFunc = stopSession

	var input *rootmodule.RootModuleManagerMockInput
	if ms.mockInput != nil {
		input = &rootmodule.RootModuleManagerMockInput{
			RootModules:        ms.mockInput.RootModules,
			TerraformExecQueue: ms.mockInput.ManagerTfExecQueue,
		}
	}

	svc := &service{
		logger:               testLogger(),
		srvCtx:               srvCtx,
		sessCtx:              sessCtx,
		stopSession:          ms.stop,
		newRootModuleManager: rootmodule.NewRootModuleManagerMock(input),
		newWatcher:           watcher.MockWatcher(),
		newWalker:            rootmodule.MockWalker,
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

func newMockSession(input *MockSessionInput) *mockSession {
	return &mockSession{mockInput: input}
}

func NewMockSession(input *MockSessionInput) session.SessionFactory {
	return newMockSession(input).new
}
