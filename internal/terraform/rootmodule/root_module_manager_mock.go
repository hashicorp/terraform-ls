package rootmodule

import (
	"context"
	"fmt"
	"log"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type RootModuleMock struct {
	TerraformExecQueue exec.MockItemDispenser
	ProviderSchemas    *tfjson.ProviderSchemas
}

type RootModuleMockFactory struct {
	rmm    map[string]*RootModuleMock
	logger *log.Logger
}

func (rmf *RootModuleMockFactory) New(ctx context.Context, dir string) (*rootModule, error) {
	rmm, ok := rmf.rmm[dir]
	if !ok {
		return nil, fmt.Errorf("unexpected root module requested: %s (%d available: %#v)", dir, len(rmf.rmm), rmf.rmm)
	}

	mock := NewRootModuleMock(ctx, rmm, dir)
	mock.SetLogger(rmf.logger)
	return mock, mock.init(ctx)
}

func NewRootModuleMock(ctx context.Context, rmm *RootModuleMock, dir string) *rootModule {
	rm := newRootModule(ctx, dir)

	md := &discovery.MockDiscovery{Path: "tf-mock"}
	rm.tfDiscoFunc = md.LookPath

	// For now, until we have better testing strategy to mimic real lock files
	rm.ignorePluginCache = true

	rm.tfNewExecutor = exec.MockExecutor(rmm.TerraformExecQueue)

	if rmm.ProviderSchemas == nil {
		rm.newSchemaStorage = schema.NewStorage
	} else {
		rm.newSchemaStorage = schema.MockStorage(rmm.ProviderSchemas)
	}

	return rm
}

func NewRootModuleManagerMock(m map[string]*RootModuleMock) RootModuleManagerFactory {
	rm := newRootModuleManager(context.Background())
	rmf := &RootModuleMockFactory{rmm: m, logger: rm.logger}
	rm.newRootModule = rmf.New

	return func(ctx context.Context) RootModuleManager {
		return rm
	}
}
