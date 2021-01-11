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

func NewModuleMock(rmm *ModuleMock, fs filesystem.Filesystem, dir string) *module {
	rm := newModule(fs, dir)

	// mock terraform discovery
	md := &discovery.MockDiscovery{Path: "tf-mock"}
	rm.tfDiscoFunc = md.LookPath

	// mock terraform executor
	rm.tfNewExecutor = rmm.TfExecFactory

	if rmm.ProviderSchemas != nil {
		rm.providerSchema = rmm.ProviderSchemas
	}

	return rm
}
