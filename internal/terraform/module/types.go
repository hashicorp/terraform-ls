package module

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/state"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfmodule "github.com/hashicorp/terraform-schema/module"
)

type File interface {
	Path() string
}

type SchemaSource struct {
	Path              string
	HumanReadablePath string
}

type ModuleFinder interface {
	ModuleByPath(path string) (Module, error)
	SchemaSourcesForModule(path string) ([]SchemaSource, error)
	ListModules() ([]Module, error)
	ModuleCalls(modPath string) ([]tfmodule.ModuleCall, error)
	CallersOfModule(modPath string) ([]Module, error)
}

type ModuleLoader func(dir string) (Module, error)

type ModuleManager interface {
	ModuleFinder

	SetLogger(logger *log.Logger)
	AddModule(modPath string) (Module, error)
	RemoveModule(modPath string) error
	EnqueueModuleOp(modPath string, opType op.OpType, deferFunc DeferFunc) error
	CancelLoading()
}

// TODO: Replace references and remove alias
type Module *state.Module

type ModuleFactory func(string) (Module, error)

type ModuleManagerFactory func(context.Context, filesystem.Filesystem, *state.ModuleStore, *state.ProviderSchemaStore) ModuleManager

type WalkerFactory func(filesystem.Filesystem, ModuleManager) *Walker

type Watcher interface {
	Start() error
	Stop() error
	SetLogger(*log.Logger)
	AddModule(string) error
	RemoveModule(string) error
	IsModuleWatched(string) bool
}
