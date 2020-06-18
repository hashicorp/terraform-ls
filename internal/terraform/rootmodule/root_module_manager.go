package rootmodule

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type rootModuleManager struct {
	rms           map[string]*rootModule
	tfExecPath    string
	tfExecTimeout time.Duration
	tfExecLogPath string
	logger        *log.Logger

	newRootModule RootModuleFactory
}

func NewRootModuleManager(ctx context.Context) RootModuleManager {
	return newRootModuleManager(ctx)
}

func newRootModuleManager(ctx context.Context) *rootModuleManager {
	rmm := &rootModuleManager{
		rms:    make(map[string]*rootModule, 0),
		logger: defaultLogger,
	}
	rmm.newRootModule = rmm.defaultRootModuleFactory
	return rmm
}

func (rmm *rootModuleManager) defaultRootModuleFactory(ctx context.Context, dir string) (*rootModule, error) {
	w := newRootModule(ctx)

	w.SetLogger(rmm.logger)

	d := &discovery.Discovery{}
	w.tfDiscoFunc = d.LookPath
	w.tfNewExecutor = exec.NewExecutor
	w.newSchemaStorage = schema.NewStorage

	w.tfExecPath = rmm.tfExecPath
	w.tfExecTimeout = rmm.tfExecTimeout
	w.tfExecLogPath = rmm.tfExecLogPath

	return w, w.init(ctx, dir)
}

func (rmm *rootModuleManager) SetTerraformExecPath(path string) {
	rmm.tfExecPath = path
}

func (rmm *rootModuleManager) SetTerraformExecLogPath(logPath string) {
	rmm.tfExecLogPath = logPath
}

func (rmm *rootModuleManager) SetTerraformExecTimeout(timeout time.Duration) {
	rmm.tfExecTimeout = timeout
}

func (rmm *rootModuleManager) SetLogger(logger *log.Logger) {
	rmm.logger = logger
}

func (rmm *rootModuleManager) AddRootModule(dir string) error {
	dir = filepath.Clean(dir)

	// TODO: Follow symlinks (requires proper test data)

	_, exists := rmm.rms[dir]
	if exists {
		return fmt.Errorf("rootModule %s was already added", dir)
	}

	w, err := rmm.newRootModule(context.Background(), dir)
	if err != nil {
		return err
	}

	rmm.rms[dir] = w

	return nil
}

func (rmm *rootModuleManager) RootModuleByPath(path string) (RootModule, error) {
	path = filepath.Clean(path)

	// TODO: Follow symlinks (requires proper test data)

	if rm, ok := rmm.rms[path]; ok {
		rmm.logger.Printf("direct root module lookup succeeded: %s", path)
		return rm, nil
	}

	dir := rootModuleDirFromFilePath(path)
	if rm, ok := rmm.rms[dir]; ok {
		rmm.logger.Printf("dir-based root module lookup succeeded: %s", dir)
		return rm, nil
	}

	for _, rm := range rmm.rms {
		rmm.logger.Printf("looking up %s in module references", dir)
		if rm.ReferencesModulePath(dir) {
			rmm.logger.Printf("module-ref-based root module lookup succeeded: %s", dir)
			return rm, nil
		}
	}

	return nil, &RootModuleNotFoundErr{path}
}

func (rmm *rootModuleManager) ParserForDir(path string) (lang.Parser, error) {
	w, err := rmm.RootModuleByPath(path)
	if err != nil {
		return nil, err
	}

	return w.Parser(), nil
}

func (rmm *rootModuleManager) TerraformExecutorForDir(path string) (*exec.Executor, error) {
	w, err := rmm.RootModuleByPath(path)
	if err != nil {
		return nil, err
	}

	return w.TerraformExecutor(), nil
}

// rootModuleDirFromPath strips known lock file paths and filenames
// to get the directory path of the relevant rootModule
func rootModuleDirFromFilePath(filePath string) string {
	pluginLockFileSuffixes := pluginLockFilePaths(string(os.PathSeparator))
	for _, s := range pluginLockFileSuffixes {
		if strings.HasSuffix(filePath, s) {
			return strings.TrimSuffix(filePath, s)
		}
	}

	moduleManifestSuffix := moduleManifestFilePath(string(os.PathSeparator))
	if strings.HasSuffix(filePath, moduleManifestSuffix) {
		return strings.TrimSuffix(filePath, moduleManifestSuffix)
	}

	return filePath
}

func (rmm *rootModuleManager) PathsToWatch() []string {
	paths := make([]string, 0)
	for _, rm := range rmm.rms {
		ptw := rm.PathsToWatch()
		if len(ptw) > 0 {
			paths = append(paths, ptw...)
		}
	}
	return paths
}
