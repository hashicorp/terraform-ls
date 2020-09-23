package rootmodule

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
)

type File interface {
	Path() string
}

type ParserFinder interface {
	ParserForDir(path string) (lang.Parser, error)
	IsParserLoaded(path string) (bool, error)
	IsSchemaLoaded(path string) (bool, error)
}

type TerraformFormatterFinder interface {
	TerraformFormatterForDir(ctx context.Context, path string) (exec.Formatter, error)
	IsTerraformLoaded(path string) (bool, error)
}

type RootModuleCandidateFinder interface {
	RootModuleCandidatesByPath(path string) RootModules
}

type RootModuleLoader func(dir string) (RootModule, error)

type RootModuleManager interface {
	ParserFinder
	TerraformFormatterFinder
	RootModuleCandidateFinder

	SetLogger(logger *log.Logger)

	SetTerraformExecPath(path string)
	SetTerraformExecLogPath(logPath string)
	SetTerraformExecTimeout(timeout time.Duration)

	AddAndStartLoadingRootModule(ctx context.Context, dir string) (RootModule, error)
	WorkerPoolSize() int
	WorkerQueueSize() int
	ListRootModules() RootModules
	PathsToWatch() []string
	RootModuleByPath(path string) (RootModule, error)
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
	LoadError() error
	StartLoading() error
	IsLoadingDone() bool
	LoadingDone() <-chan struct{}
	IsKnownPluginLockFile(path string) bool
	IsKnownModuleManifestFile(path string) bool
	PathsToWatch() []string
	UpdateSchemaCache(ctx context.Context, lockFile File) error
	ParseProviderReferences() error
	IsSchemaLoaded() bool
	UpdateModuleManifest(manifestFile File) error
	Parser() (lang.Parser, error)
	IsParserLoaded() bool
	TerraformFormatter() (exec.Formatter, error)
	IsTerraformLoaded() bool
	Modules() []ModuleRecord
}

type RootModuleFactory func(context.Context, string) (*rootModule, error)

type RootModuleManagerFactory func(tfconfig.FS) RootModuleManager

type WalkerFactory func() *Walker

type executor interface {
	SetWorkdir(string)
	SetLogger(*log.Logger)
	SetExecLogPath(string)
	SetTimeout(time.Duration)
	Version(context.Context) (string, error)
	GetExecPath() string
	FormatterForVersion(string) (exec.Formatter, error)
}

type executorFactory func(string) executor

func executorWrap(f exec.ExecutorFactory) executorFactory {
	return func(path string) executor {
		return f(path)
	}
}

func mockExecutorWrap(f exec.MockExecutorFactory) executorFactory {
	return func(path string) executor {
		return f(path)
	}
}
