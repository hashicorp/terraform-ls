package module

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type ModuleMockFactory struct {
	rmm    map[string]*ModuleMock
	logger *log.Logger
	fs     filesystem.Filesystem
}

func (rmf *ModuleMockFactory) New(ctx context.Context, dir string) (*module, error) {
	rmm, ok := rmf.rmm[dir]
	if !ok {
		return nil, fmt.Errorf("unexpected module requested: %s (%d available: %#v)", dir, len(rmf.rmm), rmf.rmm)
	}

	mock := NewModuleMock(rmm, rmf.fs, dir)
	mock.SetLogger(rmf.logger)
	return mock, mock.discoverCaches(ctx, dir)
}

type ModuleManagerMockInput struct {
	Modules           map[string]*ModuleMock
	TfExecutorFactory exec.ExecutorFactory
}

func NewModuleManagerMock(input *ModuleManagerMockInput) ModuleManagerFactory {
	return func(fs filesystem.Filesystem) ModuleManager {
		rmm := newModuleManager(fs)
		rmm.syncLoading = true

		rmf := &ModuleMockFactory{
			rmm:    make(map[string]*ModuleMock, 0),
			logger: rmm.logger,
			fs:     fs,
		}

		// mock terraform discovery
		md := &discovery.MockDiscovery{Path: "tf-mock"}
		rmm.tfDiscoFunc = md.LookPath

		// mock terraform executor
		if input != nil {
			rmm.tfNewExecutor = input.TfExecutorFactory

			if input.Modules != nil {
				rmf.rmm = input.Modules
			}
		}

		rmm.newModule = rmf.New

		return rmm
	}
}
