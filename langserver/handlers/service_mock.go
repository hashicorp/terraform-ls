package handlers

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	"github.com/hashicorp/terraform-ls/langserver/svcctl"
)

type mockService struct {
	mid exec.MockItemDispenser

	stopFunc       func()
	stopFuncCalled bool
}

func (ms *mockService) new(srvCtx context.Context) svcctl.Service {
	svcCtx, stopSvc := context.WithCancel(srvCtx)
	ms.stopFunc = stopSvc

	svc := &service{
		logger:      discardLogs,
		srvCtx:      srvCtx,
		svcCtx:      svcCtx,
		svcStopFunc: ms.stop,
		executorFunc: func(context.Context, string) *exec.Executor {
			return exec.MockExecutor(ms.mid)
		},
		ss: schema.MockStorage(nil),
	}

	return svc
}

func (ms *mockService) stop() {
	ms.stopFunc()
	ms.stopFuncCalled = true
}

func (ms *mockService) StopFuncCalled() bool {
	return ms.stopFuncCalled
}

func newMockService(mid exec.MockItemDispenser) *mockService {
	return &mockService{mid: mid}
}

func NewMock(mid exec.MockItemDispenser) svcctl.ServiceFactory {
	return newMockService(mid).new
}
