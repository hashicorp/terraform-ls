package rootmodule

import (
	"context"
	"log"
	"time"

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

type TerraformExecFinder interface {
	TerraformExecutorForDir(ctx context.Context, path string) (*exec.Executor, error)
	IsTerraformLoaded(path string) (bool, error)
}

type RootModuleCandidateFinder interface {
	RootModuleCandidatesByPath(path string) RootModules
}

type RootModuleManager interface {
	ParserFinder
	TerraformExecFinder
	RootModuleCandidateFinder

	SetLogger(logger *log.Logger)

	SetTerraformExecPath(path string)
	SetTerraformExecLogPath(logPath string)
	SetTerraformExecTimeout(timeout time.Duration)

	AddAndStartLoadingRootModule(ctx context.Context, dir string) (RootModule, error)
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
	StartLoading()
	IsLoadingDone() bool
	IsKnownPluginLockFile(path string) bool
	IsKnownModuleManifestFile(path string) bool
	PathsToWatch() []string
	UpdateSchemaCache(ctx context.Context, lockFile File) error
	IsSchemaLoaded() bool
	UpdateModuleManifest(manifestFile File) error
	Parser() (lang.Parser, error)
	IsParserLoaded() bool
	TerraformExecutor() (*exec.Executor, error)
	IsTerraformLoaded() bool
}

type RootModuleFactory func(context.Context, string) (*rootModule, error)

type RootModuleManagerFactory func() RootModuleManager

type WalkerFactory func() *Walker
