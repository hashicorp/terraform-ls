package handlers

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

type MockSessionInput struct {
	Filesystem     filesystem.Filesystem
	TerraformCalls *exec.TerraformMockCalls
}

type mockSession struct {
	mockInput *MockSessionInput

	stopFunc     func()
	stopCalled   bool
	stopCalledMu *sync.RWMutex
}

func (ms *mockSession) new(srvCtx context.Context) session.Session {
	sessCtx, stopSession := context.WithCancel(srvCtx)
	ms.stopFunc = stopSession

	var input *module.ModuleManagerMockInput
	if ms.mockInput != nil {
		input = &module.ModuleManagerMockInput{
			Logger: testLogger(),
		}
	}

	var fs filesystem.Filesystem
	if ms.mockInput != nil && ms.mockInput.Filesystem != nil {
		fs = ms.mockInput.Filesystem
	} else {
		fs = filesystem.NewFilesystem()
	}

	var tfCalls *exec.TerraformMockCalls
	if ms.mockInput != nil && ms.mockInput.TerraformCalls != nil {
		tfCalls = ms.mockInput.TerraformCalls
	}

	d := &discovery.MockDiscovery{
		Path: "tf-mock",
	}

	svc := &service{
		logger:           testLogger(),
		srvCtx:           srvCtx,
		sessCtx:          sessCtx,
		stopSession:      ms.stop,
		fs:               fs,
		newModuleManager: module.NewModuleManagerMock(input),
		newWatcher:       module.MockWatcher(),
		newWalker:        module.SyncWalker,
		tfDiscoFunc:      d.LookPath,
		tfExecFactory:    exec.NewMockExecutor(tfCalls),
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
	ms.stopCalledMu.Lock()
	defer ms.stopCalledMu.Unlock()

	ms.stopFunc()
	ms.stopCalled = true
}

func (ms *mockSession) StopFuncCalled() bool {
	ms.stopCalledMu.RLock()
	defer ms.stopCalledMu.RUnlock()

	return ms.stopCalled
}

func newMockSession(input *MockSessionInput) *mockSession {
	return &mockSession{
		mockInput:    input,
		stopCalledMu: &sync.RWMutex{},
	}
}

func NewMockSession(input *MockSessionInput) session.SessionFactory {
	return newMockSession(input).new
}
