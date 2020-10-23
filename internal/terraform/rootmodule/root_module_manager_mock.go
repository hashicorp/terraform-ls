package rootmodule

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type RootModuleMockFactory struct {
	rmm    map[string]*RootModuleMock
	logger *log.Logger
	fs     filesystem.Filesystem
}

func (rmf *RootModuleMockFactory) New(ctx context.Context, dir string) (*rootModule, error) {
	rmm, ok := rmf.rmm[dir]
	if !ok {
		return nil, fmt.Errorf("unexpected root module requested: %s (%d available: %#v)", dir, len(rmf.rmm), rmf.rmm)
	}

	mock := NewRootModuleMock(rmm, rmf.fs, dir)
	mock.SetLogger(rmf.logger)
	return mock, mock.discoverCaches(ctx, dir)
}

type RootModuleManagerMockInput struct {
	RootModules       map[string]*RootModuleMock
	TfExecutorFactory exec.ExecutorFactory
}

func NewRootModuleManagerMock(input *RootModuleManagerMockInput) RootModuleManagerFactory {
	return func(fs filesystem.Filesystem) RootModuleManager {
		rmm := newRootModuleManager(fs)
		rmm.syncLoading = true

		rmf := &RootModuleMockFactory{
			rmm:    make(map[string]*RootModuleMock, 0),
			logger: rmm.logger,
			fs:     fs,
		}

		// mock terraform discovery
		md := &discovery.MockDiscovery{Path: "tf-mock"}
		rmm.tfDiscoFunc = md.LookPath

		// mock terraform executor
		if input != nil {
			rmm.tfNewExecutor = input.TfExecutorFactory

			if input.RootModules != nil {
				rmf.rmm = input.RootModules
			}
		}

		rmm.newRootModule = rmf.New

		return rmm
	}
}
