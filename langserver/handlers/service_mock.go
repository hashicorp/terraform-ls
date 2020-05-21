package handlers

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	"github.com/hashicorp/terraform-ls/langserver/session"
)

type mockSession struct {
	mid exec.MockItemDispenser

	stopFunc       func()
	stopFuncCalled bool
}

func (ms *mockSession) new(srvCtx context.Context) session.Session {
	sessCtx, stopSession := context.WithCancel(srvCtx)
	ms.stopFunc = stopSession
	d := discovery.MockDiscovery{Path: "mock-tf"}

	svc := &service{
		logger:      discardLogs,
		srvCtx:      srvCtx,
		sessCtx:     sessCtx,
		stopSession: ms.stop,
		executorFunc: func(context.Context, string) *exec.Executor {
			return exec.MockExecutor(ms.mid)
		},
		tfDiscoFunc: d.LookPath,
		ss:          schema.MockStorage(nil),
	}

	return svc
}

func (ms *mockSession) stop() {
	ms.stopFunc()
	ms.stopFuncCalled = true
}

func (ms *mockSession) StopFuncCalled() bool {
	return ms.stopFuncCalled
}

func newMockSession(mid exec.MockItemDispenser) *mockSession {
	return &mockSession{mid: mid}
}

func NewMock(mid exec.MockItemDispenser) session.SessionFactory {
	return newMockSession(mid).new
}
