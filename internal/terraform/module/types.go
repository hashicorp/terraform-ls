package module

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type File interface {
	Path() string
}

type TerraformFormatterFinder interface {
	TerraformFormatterForDir(ctx context.Context, path string) (exec.Formatter, error)
	HasTerraformDiscoveryFinished(path string) (bool, error)
	IsTerraformAvailable(path string) (bool, error)
}

type ModuleFinder interface {
	ModuleCandidatesByPath(path string) Modules
	ModuleByPath(path string) (Module, error)
	SchemaForPath(path string) (*schema.BodySchema, error)
}

type ModuleLoader func(dir string) (Module, error)

type ModuleManager interface {
	ModuleFinder
	TerraformFormatterFinder

	SetLogger(logger *log.Logger)

	SetTerraformExecPath(path string)
	SetTerraformExecLogPath(logPath string)
	SetTerraformExecTimeout(timeout time.Duration)

	InitAndUpdateModule(ctx context.Context, dir string) (Module, error)
	AddAndStartLoadingModule(ctx context.Context, dir string) (Module, error)
	WorkerPoolSize() int
	WorkerQueueSize() int
	ListModules() Modules
	PathsToWatch() []string
	CancelLoading()
}

type Modules []Module

func (mods Modules) Paths() []string {
	paths := make([]string, len(mods))
	for i, mod := range mods {
		paths[i] = mod.Path()
	}
	return paths
}

type Module interface {
	Path() string
	MatchesPath(path string) bool
	LoadError() error
	StartLoading() error
	IsLoadingDone() bool
	LoadingDone() <-chan struct{}
	IsKnownPluginLockFile(path string) bool
	IsKnownModuleManifestFile(path string) bool
	PathsToWatch() []string
	UpdateProviderSchemaCache(ctx context.Context, lockFile File) error
	IsProviderSchemaLoaded() bool
	UpdateModuleManifest(manifestFile File) error
	Decoder() (*decoder.Decoder, error)
	DecoderWithSchema(*schema.BodySchema) (*decoder.Decoder, error)
	MergedSchema() (*schema.BodySchema, error)
	IsParsed() bool
	ParseFiles() error
	ParsedDiagnostics() map[string]hcl.Diagnostics
	TerraformFormatter() (exec.Formatter, error)
	HasTerraformDiscoveryFinished() bool
	IsTerraformAvailable() bool
	ExecuteTerraformInit(ctx context.Context) error
	ExecuteTerraformValidate(ctx context.Context) (map[string]hcl.Diagnostics, error)
	Modules() []ModuleRecord
	HumanReadablePath(string) string
	WasInitialized() (bool, error)
}

type ModuleFactory func(context.Context, string) (*module, error)

type ModuleManagerFactory func(filesystem.Filesystem) ModuleManager

type WalkerFactory func() *Walker
