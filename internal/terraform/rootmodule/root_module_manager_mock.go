package rootmodule

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type RootModuleMockFactory struct {
	rmm    map[string]*RootModuleMock
	logger *log.Logger
}

func (rmf *RootModuleMockFactory) New(ctx context.Context, dir string) (*rootModule, error) {
	rmm, ok := rmf.rmm[dir]
	if !ok {
		return nil, fmt.Errorf("unexpected root module requested: %s (%d available: %#v)", dir, len(rmf.rmm), rmf.rmm)
	}

	mock := NewRootModuleMock(rmm, dir)
	mock.SetLogger(rmf.logger)
	return mock, mock.discoverCaches(ctx, dir)
}

type RootModuleManagerMockInput struct {
	RootModules        map[string]*RootModuleMock
	TerraformExecQueue exec.MockItemDispenser
}

func NewRootModuleManagerMock(input *RootModuleManagerMockInput) RootModuleManagerFactory {
	rmm := newRootModuleManager()
	rmm.syncLoading = true

	rmf := &RootModuleMockFactory{
		rmm:    make(map[string]*RootModuleMock, 0),
		logger: rmm.logger,
	}

	// mock terraform discovery
	md := &discovery.MockDiscovery{Path: "tf-mock"}
	rmm.tfDiscoFunc = md.LookPath

	// mock terraform executor
	if input != nil {
		rmm.tfNewExecutor = exec.MockExecutor(input.TerraformExecQueue)

		if input.RootModules != nil {
			rmf.rmm = input.RootModules
		}
	}

	rmm.newRootModule = rmf.New

	return func() RootModuleManager {
		return rmm
	}
}
