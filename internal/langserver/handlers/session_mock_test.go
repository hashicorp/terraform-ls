package handlers

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/creachadair/jrpc2/handler"
	"github.com/hashicorp/terraform-ls/internal/langserver/session"
	"github.com/hashicorp/terraform-ls/internal/registry"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/walker"
)

type MockSessionInput struct {
	TerraformCalls     *exec.TerraformMockCalls
	AdditionalHandlers map[string]handler.Func
	StateStore         *state.StateStore
	WalkerCollector    *walker.WalkerCollector
	RegistryServer     *httptest.Server
}

type mockSession struct {
	mockInput      *MockSessionInput
	registryServer *httptest.Server

	stopFunc     func()
	stopCalled   bool
	stopCalledMu *sync.RWMutex
}

func (ms *mockSession) new(srvCtx context.Context) session.Session {
	sessCtx, stopSession := context.WithCancel(srvCtx)
	ms.stopFunc = stopSession

	var handlers map[string]handler.Func
	var stateStore *state.StateStore
	var walkerCollector *walker.WalkerCollector
	if ms.mockInput != nil {
		stateStore = ms.mockInput.StateStore
		walkerCollector = ms.mockInput.WalkerCollector
		handlers = ms.mockInput.AdditionalHandlers
		ms.registryServer = ms.mockInput.RegistryServer
	}

	var tfCalls *exec.TerraformMockCalls
	if ms.mockInput != nil && ms.mockInput.TerraformCalls != nil {
		tfCalls = ms.mockInput.TerraformCalls
	}

	d := &discovery.MockDiscovery{
		Path: "tf-mock",
	}

	regClient := registry.NewClient()
	if ms.registryServer == nil {
		ms.registryServer = defaultRegistryServer()
	}
	ms.registryServer.Start()

	regClient.BaseURL = ms.registryServer.URL

	svc := &service{
		logger:             testLogger(),
		srvCtx:             srvCtx,
		sessCtx:            sessCtx,
		stopSession:        ms.stop,
		tfDiscoFunc:        d.LookPath,
		tfExecFactory:      exec.NewMockExecutor(tfCalls),
		additionalHandlers: handlers,
		stateStore:         stateStore,
		walkerCollector:    walkerCollector,
		registryClient:     regClient,
	}

	return svc
}

func defaultRegistryServer() *httptest.Server {
	return httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unexpected Registry API request", 500)
	}))
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	return log.New(ioutil.Discard, "", 0)
}

func (ms *mockSession) stop() {
	ms.registryServer.Close()

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
