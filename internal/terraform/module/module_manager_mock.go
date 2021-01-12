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
	mmocks map[string]*ModuleMock
	logger *log.Logger
	fs     filesystem.Filesystem
}

func (mmf *ModuleMockFactory) New(ctx context.Context, dir string) (*module, error) {
	mmocks, ok := mmf.mmocks[dir]
	if !ok {
		return nil, fmt.Errorf("unexpected module requested: %s (%d available: %#v)", dir, len(mmf.mmocks), mmf.mmocks)
	}

	mock := NewModuleMock(mmocks, mmf.fs, dir)
	mock.SetLogger(mmf.logger)
	return mock, mock.discoverCaches(ctx, dir)
}

type ModuleManagerMockInput struct {
	Modules           map[string]*ModuleMock
	TfExecutorFactory exec.ExecutorFactory
}

func NewModuleManagerMock(input *ModuleManagerMockInput) ModuleManagerFactory {
	return func(fs filesystem.Filesystem) ModuleManager {
		mm := newModuleManager(fs)
		mm.syncLoading = true

		mmf := &ModuleMockFactory{
			mmocks: make(map[string]*ModuleMock, 0),
			logger: mm.logger,
			fs:     fs,
		}

		// mock terraform discovery
		md := &discovery.MockDiscovery{Path: "tf-mock"}
		mm.tfDiscoFunc = md.LookPath

		// mock terraform executor
		if input != nil {
			mm.tfNewExecutor = input.TfExecutorFactory

			if input.Modules != nil {
				mmf.mmocks = input.Modules
			}
		}

		mm.newModule = mmf.New

		return mm
	}
}
