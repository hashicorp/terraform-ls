package module

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type moduleManager struct {
	rms        []*module
	newModule  ModuleFactory
	filesystem filesystem.Filesystem

	syncLoading bool
	workerPool  *workerpool.WorkerPool
	logger      *log.Logger

	// terraform discovery
	tfDiscoFunc discovery.DiscoveryFunc

	// terraform executor
	tfNewExecutor exec.ExecutorFactory
	tfExecPath    string
	tfExecTimeout time.Duration
	tfExecLogPath string
}

func NewModuleManager(fs filesystem.Filesystem) ModuleManager {
	return newModuleManager(fs)
}

func newModuleManager(fs filesystem.Filesystem) *moduleManager {
	d := &discovery.Discovery{}

	defaultSize := 3 * runtime.NumCPU()
	wp := workerpool.New(defaultSize)

	rmm := &moduleManager{
		rms:           make([]*module, 0),
		filesystem:    fs,
		workerPool:    wp,
		logger:        defaultLogger,
		tfDiscoFunc:   d.LookPath,
		tfNewExecutor: exec.NewExecutor,
	}
	rmm.newModule = rmm.defaultModuleFactory
	return rmm
}

func (rmm *moduleManager) WorkerPoolSize() int {
	return rmm.workerPool.Size()
}

func (rmm *moduleManager) WorkerQueueSize() int {
	return rmm.workerPool.WaitingQueueSize()
}

func (rmm *moduleManager) defaultModuleFactory(ctx context.Context, dir string) (*module, error) {
	rm := newModule(rmm.filesystem, dir)

	rm.SetLogger(rmm.logger)

	d := &discovery.Discovery{}
	rm.tfDiscoFunc = d.LookPath
	rm.tfNewExecutor = exec.NewExecutor

	rm.tfExecPath = rmm.tfExecPath
	rm.tfExecTimeout = rmm.tfExecTimeout
	rm.tfExecLogPath = rmm.tfExecLogPath

	return rm, rm.discoverCaches(ctx, dir)
}

func (rmm *moduleManager) SetTerraformExecPath(path string) {
	rmm.tfExecPath = path
}

func (rmm *moduleManager) SetTerraformExecLogPath(logPath string) {
	rmm.tfExecLogPath = logPath
}

func (rmm *moduleManager) SetTerraformExecTimeout(timeout time.Duration) {
	rmm.tfExecTimeout = timeout
}

func (rmm *moduleManager) SetLogger(logger *log.Logger) {
	rmm.logger = logger
}

func (rmm *moduleManager) InitAndUpdateModule(ctx context.Context, dir string) (Module, error) {
	rm, err := rmm.ModuleByPath(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get module: %+v", err)
	}

	if err := rm.ExecuteTerraformInit(ctx); err != nil {
		return nil, fmt.Errorf("failed to init module: %+v", err)
	}

	m := rm.(*module)
	m.discoverCaches(ctx, dir)
	return rm, m.UpdateProviderSchemaCache(ctx, m.pluginLockFile)
}

func (rmm *moduleManager) AddAndStartLoadingModule(ctx context.Context, dir string) (Module, error) {
	dir = filepath.Clean(dir)

	// TODO: Follow symlinks (requires proper test data)

	if _, ok := rmm.moduleByPath(dir); ok {
		return nil, fmt.Errorf("module %s was already added", dir)
	}

	rm, err := rmm.newModule(context.Background(), dir)
	if err != nil {
		return nil, err
	}

	rmm.rms = append(rmm.rms, rm)

	if rmm.syncLoading {
		rmm.logger.Printf("synchronously loading module %s", dir)
		return rm, rm.load(ctx)
	}

	rmm.logger.Printf("asynchronously loading module %s", dir)
	rmm.workerPool.Submit(func() {
		rm := rm
		err := rm.load(context.Background())
		rm.setLoadErr(err)
	})

	return rm, nil
}

func (rmm *moduleManager) SchemaForPath(path string) (*schema.BodySchema, error) {
	candidates := rmm.ModuleCandidatesByPath(path)
	for _, rm := range candidates {
		schema, err := rm.MergedSchema()
		if err != nil {
			rmm.logger.Printf("failed to merge schema for %s: %s", rm.Path(), err)
			continue
		}
		if schema != nil {
			rmm.logger.Printf("found schema for %s at %s", path, rm.Path())
			return schema, nil
		}
	}

	rm, err := rmm.ModuleByPath(path)
	if err != nil {
		return nil, err
	}

	return rm.MergedSchema()
}

func (rmm *moduleManager) moduleByPath(dir string) (*module, bool) {
	for _, rm := range rmm.rms {
		if pathEquals(rm.Path(), dir) {
			return rm, true
		}
	}
	return nil, false
}

// ModuleCandidatesByPath finds any initialized modules
func (rmm *moduleManager) ModuleCandidatesByPath(path string) Modules {
	path = filepath.Clean(path)

	candidates := make([]Module, 0)

	// TODO: Follow symlinks (requires proper test data)

	rm, foundPath := rmm.moduleByPath(path)
	if foundPath {
		inited, _ := rm.WasInitialized()
		if inited {
			candidates = append(candidates, rm)
		}
	}

	if !foundPath {
		dir := trimLockFilePath(path)
		rm, ok := rmm.moduleByPath(dir)
		if ok {
			inited, _ := rm.WasInitialized()
			if inited {
				candidates = append(candidates, rm)
			}
		}
	}

	for _, rm := range rmm.rms {
		if rm.ReferencesModulePath(path) {
			candidates = append(candidates, rm)
		}
	}

	return candidates
}

func (rmm *moduleManager) ListModules() Modules {
	modules := make([]Module, 0)
	for _, rm := range rmm.rms {
		modules = append(modules, rm)
	}
	return modules
}
func (rmm *moduleManager) ModuleByPath(path string) (Module, error) {
	path = filepath.Clean(path)

	if rm, ok := rmm.moduleByPath(path); ok {
		return rm, nil
	}

	dir := trimLockFilePath(path)

	if rm, ok := rmm.moduleByPath(dir); ok {
		return rm, nil
	}

	return nil, &ModuleNotFoundErr{path}
}

func (rmm *moduleManager) IsProviderSchemaLoaded(path string) (bool, error) {
	rm, err := rmm.ModuleByPath(path)
	if err != nil {
		return false, err
	}

	return rm.IsProviderSchemaLoaded(), nil
}

func (rmm *moduleManager) TerraformFormatterForDir(ctx context.Context, path string) (exec.Formatter, error) {
	rm, err := rmm.ModuleByPath(path)
	if err != nil {
		if IsModuleNotFound(err) {
			return rmm.newTerraformFormatter(ctx, path)
		}
		return nil, err
	}

	return rm.TerraformFormatter()
}

func (rmm *moduleManager) newTerraformFormatter(ctx context.Context, workDir string) (exec.Formatter, error) {
	tfPath := rmm.tfExecPath
	if tfPath == "" {
		var err error
		tfPath, err = rmm.tfDiscoFunc()
		if err != nil {
			return nil, err
		}
	}

	tf, err := rmm.tfNewExecutor(workDir, tfPath)
	if err != nil {
		return nil, err
	}

	tf.SetLogger(rmm.logger)

	if rmm.tfExecLogPath != "" {
		tf.SetExecLogPath(rmm.tfExecLogPath)
	}

	if rmm.tfExecTimeout != 0 {
		tf.SetTimeout(rmm.tfExecTimeout)
	}

	version, _, err := tf.Version(ctx)
	if err != nil {
		return nil, err
	}
	rmm.logger.Printf("Terraform version %s found at %s (alternative)", version, tf.GetExecPath())

	return tf.Format, nil
}

func (rmm *moduleManager) IsTerraformAvailable(path string) (bool, error) {
	rm, err := rmm.ModuleByPath(path)
	if err != nil {
		return false, err
	}

	return rm.IsTerraformAvailable(), nil
}

func (rmm *moduleManager) HasTerraformDiscoveryFinished(path string) (bool, error) {
	rm, err := rmm.ModuleByPath(path)
	if err != nil {
		return false, err
	}

	return rm.HasTerraformDiscoveryFinished(), nil
}

func (rmm *moduleManager) CancelLoading() {
	for _, rm := range rmm.rms {
		rmm.logger.Printf("cancelling loading for %s", rm.Path())
		rm.CancelLoading()
		rmm.logger.Printf("loading cancelled for %s", rm.Path())
	}
	rmm.workerPool.Stop()
}

// trimLockFilePath strips known lock file paths and filenames
// to get the directory path of the relevant module
func trimLockFilePath(filePath string) string {
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

func (rmm *moduleManager) PathsToWatch() []string {
	paths := make([]string, 0)
	for _, rm := range rmm.rms {
		ptw := rm.PathsToWatch()
		if len(ptw) > 0 {
			paths = append(paths, ptw...)
		}
	}
	return paths
}

// NewModuleLoader allows adding & loading modules
// with a given context. This can be passed down to any handler
// which itself will have short-lived context
// therefore couldn't finish loading the module asynchronously
// after it responds to the client
func NewModuleLoader(ctx context.Context, rmm ModuleManager) ModuleLoader {
	return func(dir string) (Module, error) {
		return rmm.AddAndStartLoadingModule(ctx, dir)
	}
}
