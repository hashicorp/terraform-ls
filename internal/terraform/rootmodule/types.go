package rootmodule

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

type RootModuleFinder interface {
	RootModuleCandidatesByPath(path string) RootModules
	RootModuleByPath(path string) (RootModule, error)
	SchemaForPath(path string) (*schema.BodySchema, error)
}

type RootModuleLoader func(dir string) (RootModule, error)

type RootModuleManager interface {
	RootModuleFinder
	TerraformFormatterFinder

	SetLogger(logger *log.Logger)

	SetTerraformExecPath(path string)
	SetTerraformExecLogPath(logPath string)
	SetTerraformExecTimeout(timeout time.Duration)

	AddAndStartLoadingRootModule(ctx context.Context, dir string) (RootModule, error)
	WorkerPoolSize() int
	WorkerQueueSize() int
	ListRootModules() RootModules
	PathsToWatch() []string
	CancelLoading()
}

type RootModules []RootModule

func (rms RootModules) Paths() []string {
	paths := make([]string, len(rms))
	for i, rm := range rms {
		paths[i] = rm.Path()
	}
	return paths
}

type RootModule interface {
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
	IsCoreSchemaLoaded() bool
	TerraformFormatter() (exec.Formatter, error)
	HasTerraformDiscoveryFinished() bool
	IsTerraformAvailable() bool
	Modules() []ModuleRecord
}

type RootModuleFactory func(context.Context, string) (*rootModule, error)

type RootModuleManagerFactory func(filesystem.Filesystem) RootModuleManager

type WalkerFactory func() *Walker
