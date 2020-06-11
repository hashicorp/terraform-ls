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
	rm, ok := rmf.rmm[dir]
	if !ok {
		return nil, fmt.Errorf("unexpected root module requested: %s (%d available: %#v)", dir, len(rmf.rmm), rmf.rmm)
	}

	w := newRootModule(ctx)
	w.SetLogger(rmf.logger)

	md := &discovery.MockDiscovery{Path: "tf-mock"}
	w.tfDiscoFunc = md.LookPath

	// For now, until we have better testing strategy to mimic real lock files
	w.ignorePluginCache = true

	w.tfNewExecutor = exec.MockExecutor(rm.TerraformExecQueue)

	if rm.ProviderSchemas == nil {
		w.newSchemaStorage = schema.MockStorage(rm.ProviderSchemas)
	} else {
		w.newSchemaStorage = schema.NewStorage
	}

	return w, w.init(ctx, dir)
}

func NewRootModuleManagerMock(m map[string]*RootModuleMock) RootModuleManagerFactory {
	rm := newRootModuleManager(context.Background())
	rmf := &RootModuleMockFactory{rmm: m, logger: rm.logger}
	rm.newRootModule = rmf.New

	return func(ctx context.Context) RootModuleManager {
		return rm
	}
}
