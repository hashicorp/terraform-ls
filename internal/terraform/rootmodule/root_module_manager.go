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

	// Many root modules can be added in a short time period
	// Running init asynchronously makes it more efficient
	// and prevents flooding the user with errors
	// which may not be relevant for them until they actually
	// open the affected directory
	go func(ctx context.Context, rm *rootModule) {
		rm.lastErr = rm.init(ctx)
	}(ctx, rm)

	return rm, nil
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

	if _, ok := rmm.rootModuleByPath(dir); ok {
		return fmt.Errorf("root module %s was already added", dir)
	}

	rm, err := rmm.newRootModule(context.Background(), dir)
	if err != nil {
		return err
	}

	rmm.rms = append(rmm.rms, rm)

	return nil
}

func (rmm *rootModuleManager) rootModuleByPath(dir string) (*rootModule, bool) {
	for _, rm := range rmm.rms {
		if rm.Path() == dir {
			return rm, true
		}
	}
	return nil, false
}

func (rmm *rootModuleManager) RootModuleCandidatesByPath(path string) RootModules {
	path = filepath.Clean(path)

	// TODO: Follow symlinks (requires proper test data)

	if rm, ok := rmm.rootModuleByPath(path); ok {
		rmm.logger.Printf("direct root module lookup succeeded: %s", path)
		return []RootModule{rm}
	}

	dir := rootModuleDirFromFilePath(path)
	if rm, ok := rmm.rootModuleByPath(dir); ok {
		rmm.logger.Printf("dir-based root module lookup succeeded: %s", dir)
		return []RootModule{rm}
	}

	candidates := make([]RootModule, 0)
	for _, rm := range rmm.rms {
		rmm.logger.Printf("looking up %s in module references of %s", dir, rm.Path())
		if rm.ReferencesModulePath(dir) {
			rmm.logger.Printf("module-ref-based root module lookup succeeded: %s", dir)
			candidates = append(candidates, rm)
		}
	}

	return candidates
}

func (rmm *rootModuleManager) RootModuleByPath(path string) (RootModule, error) {
	candidates := rmm.RootModuleCandidatesByPath(path)
	if len(candidates) > 0 {
		return candidates[0], nil
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
