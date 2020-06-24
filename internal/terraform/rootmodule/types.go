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
	TerraformExecutorForDir(path string) (*exec.Executor, error)
}

type RootModuleCandidateFinder interface {
	RootModuleCandidatesByPath(path string) []string
}

type RootModuleManager interface {
	ParserFinder
	TerraformExecFinder
	RootModuleCandidateFinder

	SetLogger(logger *log.Logger)
	SetTerraformExecPath(path string)
	SetTerraformExecLogPath(logPath string)
	SetTerraformExecTimeout(timeout time.Duration)
	AddRootModule(dir string) error
	PathsToWatch() []string
	RootModuleByPath(path string) (RootModule, error)
}

type RootModule interface {
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
