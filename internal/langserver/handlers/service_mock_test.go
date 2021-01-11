package handlers

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/hashicorp/terraform-ls/internal/watcher"
)

type MockSessionInput struct {
	RootModules       map[string]*module.RootModuleMock
	Filesystem        filesystem.Filesystem
	TfExecutorFactory exec.ExecutorFactory
}

type mockSession struct {
	mockInput *MockSessionInput

	stopFunc       func()
	stopFuncCalled bool
}

func (ms *mockSession) new(srvCtx context.Context) session.Session {
	sessCtx, stopSession := context.WithCancel(srvCtx)
	ms.stopFunc = stopSession

	var input *module.RootModuleManagerMockInput
	if ms.mockInput != nil {
		input = &module.RootModuleManagerMockInput{
			RootModules:       ms.mockInput.RootModules,
			TfExecutorFactory: ms.mockInput.TfExecutorFactory,
		}
	}

	var fs filesystem.Filesystem
	if ms.mockInput != nil && ms.mockInput.Filesystem != nil {
		fs = ms.mockInput.Filesystem
	} else {
		fs = filesystem.NewFilesystem()
	}

	svc := &service{
		logger:               testLogger(),
		srvCtx:               srvCtx,
		sessCtx:              sessCtx,
		stopSession:          ms.stop,
		fs:                   fs,
		newRootModuleManager: module.NewRootModuleManagerMock(input),
		newWatcher:           watcher.MockWatcher(),
		newWalker:            module.MockWalker,
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
