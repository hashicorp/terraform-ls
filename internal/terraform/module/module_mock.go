package module

import (
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type ModuleMock struct {
	TfExecFactory   exec.ExecutorFactory
	ProviderSchemas *tfjson.ProviderSchemas
}

func NewModuleMock(modMock *ModuleMock, fs filesystem.Filesystem, dir string) *module {
	module := newModule(fs, dir)

	// mock terraform discovery
	md := &discovery.MockDiscovery{Path: "tf-mock"}
	module.tfDiscoFunc = md.LookPath

	// mock terraform executor
	module.tfNewExecutor = modMock.TfExecFactory

	if modMock.ProviderSchemas != nil {
		module.providerSchema = modMock.ProviderSchemas
	}

	return module
}
