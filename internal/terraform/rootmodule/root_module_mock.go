package rootmodule

import (
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type RootModuleMock struct {
	TfExecFactory   exec.ExecutorFactory
	ProviderSchemas *tfjson.ProviderSchemas
}

func NewRootModuleMock(rmm *RootModuleMock, fs filesystem.Filesystem, dir string) *rootModule {
	rm := newRootModule(fs, dir)

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
