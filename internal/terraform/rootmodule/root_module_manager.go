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
	rms           []*rootModule
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
		rms:    make([]*rootModule, 0),
		logger: defaultLogger,
	}
	rmm.newRootModule = rmm.defaultRootModuleFactory
	return rmm
}

func (rmm *rootModuleManager) defaultRootModuleFactory(ctx context.Context, dir string) (*rootModule, error) {
	rm := newRootModule(ctx, dir)

	rm.SetLogger(rmm.logger)

	d := &discovery.Discovery{}
	rm.tfDiscoFunc = d.LookPath
	rm.tfNewExecutor = exec.NewExecutor
	rm.newSchemaStorage = schema.NewStorage

	rm.tfExecPath = rmm.tfExecPath
	rm.tfExecTimeout = rmm.tfExecTimeout
	rm.tfExecLogPath = rmm.tfExecLogPath

	return rm, rm.init(ctx)
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

	if rmm.exists(dir) {
		return fmt.Errorf("root module %s was already added", dir)
	}

	rm, err := rmm.newRootModule(context.Background(), dir)
	if err != nil {
		return err
	}

	rmm.rms = append(rmm.rms, rm)

	return nil
}

func (rmm *rootModuleManager) exists(dir string) bool {
	for _, rm := range rmm.rms {
		if rm.Path() == dir {
			return true
		}
	}
	return false
}

func (rmm *rootModuleManager) rootModuleByPath(dir string) *rootModule {
	for _, rm := range rmm.rms {
		if rm.Path() == dir {
			return rm
		}
	}
	return nil
}

func (rmm *rootModuleManager) RootModuleCandidatesByPath(path string) []string {
	path = filepath.Clean(path)

	// TODO: Follow symlinks (requires proper test data)

	if rmm.exists(path) {
		rmm.logger.Printf("direct root module lookup succeeded: %s", path)
		return []string{path}
	}

	dir := rootModuleDirFromFilePath(path)
	if rmm.exists(dir) {
		rmm.logger.Printf("dir-based root module lookup succeeded: %s", dir)
		return []string{dir}
	}

	candidates := make([]string, 0)
	for _, rm := range rmm.rms {
		rmm.logger.Printf("looking up %s in module references of %s", dir, rm.Path())
		if rm.ReferencesModulePath(dir) {
			rmm.logger.Printf("module-ref-based root module lookup succeeded: %s", dir)
			candidates = append(candidates, rm.Path())
		}
	}

	return candidates
}

func (rmm *rootModuleManager) RootModuleByPath(path string) (RootModule, error) {
	candidates := rmm.RootModuleCandidatesByPath(path)
	if len(candidates) > 0 {
		firstMatch := candidates[0]
		if !rmm.exists(firstMatch) {
			return nil, fmt.Errorf("Discovered root module %s not available,"+
				" this is most likely a bug, please report it", firstMatch)
		}
		return rmm.rootModuleByPath(firstMatch), nil
	}

	return nil, &RootModuleNotFoundErr{path}
}

func (rmm *rootModuleManager) ParserForDir(path string) (lang.Parser, error) {
	rm, err := rmm.RootModuleByPath(path)
	if err != nil {
		return nil, err
	}

	return rm.Parser(), nil
}

func (rmm *rootModuleManager) TerraformExecutorForDir(ctx context.Context, path string) (*exec.Executor, error) {
	rm, err := rmm.RootModuleByPath(path)
	if err != nil && IsRootModuleNotFound(err) {
		return rmm.terraformExecutorForDir(ctx, path)
	}

	return rm.TerraformExecutor(), nil
}

func (rmm *rootModuleManager) terraformExecutorForDir(ctx context.Context, dir string) (*exec.Executor, error) {
	tfPath := rmm.tfExecPath
	if tfPath == "" {
		var err error
		d := &discovery.Discovery{}
		tfPath, err = d.LookPath()
		if err != nil {
			return nil, err
		}
	}

	tf := exec.NewExecutor(ctx, tfPath)

	tf.SetWorkdir(dir)
	tf.SetLogger(rmm.logger)

	if rmm.tfExecLogPath != "" {
		tf.SetExecLogPath(rmm.tfExecLogPath)
	}

	if rmm.tfExecTimeout != 0 {
		tf.SetTimeout(rmm.tfExecTimeout)
	}

	return tf, nil
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
