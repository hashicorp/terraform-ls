package module

import (
	"context"
	"log"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
)

type File interface {
	Path() string
}

type SchemaSource interface {
	// module specific methods
	Path() string
	HumanReadablePath(string) string

	ProviderSchema() (*tfjson.ProviderSchemas, error)
	TerraformVersion() (*version.Version, error)
	ProviderVersions() map[string]*version.Version
}

type ModuleFinder interface {
	ModuleByPath(path string) (Module, error)
	SchemaForModule(path string) (*schema.BodySchema, error)
	SchemaSourcesForModule(path string) ([]SchemaSource, error)
	ListModules() []Module
}

type ModuleLoader func(dir string) (Module, error)

type ModuleManager interface {
	ModuleFinder

	SetLogger(logger *log.Logger)
	AddModule(modPath string) (Module, error)
	EnqueueModuleOp(modPath string, opType OpType) error
	EnqueueModuleOpWait(modPath string, opType OpType) error
	CancelLoading()
}

type Module interface {
	Path() string
	HumanReadablePath(string) string
	MatchesPath(path string) bool
	HasOpenFiles() bool

	TerraformExecPath() string
	TerraformVersion() (*version.Version, error)
	ProviderVersions() map[string]*version.Version
	ProviderSchema() (*tfjson.ProviderSchemas, error)
	ModuleManifest() (*datadir.ModuleManifest, error)

	TerraformVersionState() OpState
	ProviderSchemaState() OpState

	ParsedFiles() (map[string]*hcl.File, error)
	Diagnostics() map[string]hcl.Diagnostics
	ModuleCalls() []datadir.ModuleRecord
}

type ModuleFactory func(string) (*module, error)

type ModuleManagerFactory func(context.Context, filesystem.Filesystem) ModuleManager

type WalkerFactory func(filesystem.Filesystem, ModuleManager) *Walker

type Watcher interface {
	Start() error
	Stop() error
	SetLogger(*log.Logger)
	AddModule(string) error
	IsModuleWatched(string) bool
}
