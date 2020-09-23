package rootmodule

import (
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type RootModuleMock struct {
	TerraformExecQueue exec.MockItemDispenser
	ProviderSchemas    *tfjson.ProviderSchemas
}

func NewRootModuleMock(rmm *RootModuleMock, dir string) *rootModule {
	fs := filesystem.NewFilesystem()
	rm := newRootModule(fs, dir)

	// mock terraform discovery
	md := &discovery.MockDiscovery{Path: "tf-mock"}
	rm.tfDiscoFunc = md.LookPath

	// mock terraform executor
	rm.tfNewExecutor = mockExecutorWrap(exec.MockExecutor(rmm.TerraformExecQueue))

	if rmm.ProviderSchemas == nil {
		rm.newSchemaStorage = schema.NewStorageForVersion
	} else {
		rm.newSchemaStorage = schema.NewMockStorage(rmm.ProviderSchemas)
	}

	return rm
}
