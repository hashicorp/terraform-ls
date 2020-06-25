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
}

type TerraformExecFinder interface {
	TerraformExecutorForDir(ctx context.Context, path string) (*exec.Executor, error)
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
	AddRootModule(dir string) (RootModule, error)
	PathsToWatch() []string
	RootModuleByPath(path string) (RootModule, error)
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
	IsKnownPluginLockFile(path string) bool
	IsKnownModuleManifestFile(path string) bool
	PathsToWatch() []string
	UpdatePluginCache(lockFile File) error
	UpdateModuleManifest(manifestFile File) error
	Parser() lang.Parser
	TerraformExecutor() *exec.Executor
}

type RootModuleFactory func(context.Context, string) (*rootModule, error)

type RootModuleManagerFactory func(context.Context) RootModuleManager

type WalkerFactory func() *Walker
