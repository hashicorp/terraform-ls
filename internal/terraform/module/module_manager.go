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
	modules    []*module
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

	mm := &moduleManager{
		modules:       make([]*module, 0),
		filesystem:    fs,
		workerPool:    wp,
		logger:        defaultLogger,
		tfDiscoFunc:   d.LookPath,
		tfNewExecutor: exec.NewExecutor,
	}
	mm.newModule = mm.defaultModuleFactory
	return mm
}

func (mm *moduleManager) WorkerPoolSize() int {
	return mm.workerPool.Size()
}

func (mm *moduleManager) WorkerQueueSize() int {
	return mm.workerPool.WaitingQueueSize()
}

func (mm *moduleManager) defaultModuleFactory(ctx context.Context, dir string) (*module, error) {
	mod := newModule(mm.filesystem, dir)

	mod.SetLogger(mm.logger)

	d := &discovery.Discovery{}
	mod.tfDiscoFunc = d.LookPath
	mod.tfNewExecutor = exec.NewExecutor

	mod.tfExecPath = mm.tfExecPath
	mod.tfExecTimeout = mm.tfExecTimeout
	mod.tfExecLogPath = mm.tfExecLogPath

	return mod, mod.discoverCaches(ctx, dir)
}

func (mm *moduleManager) SetTerraformExecPath(path string) {
	mm.tfExecPath = path
}

func (mm *moduleManager) SetTerraformExecLogPath(logPath string) {
	mm.tfExecLogPath = logPath
}

func (mm *moduleManager) SetTerraformExecTimeout(timeout time.Duration) {
	mm.tfExecTimeout = timeout
}

func (mm *moduleManager) SetLogger(logger *log.Logger) {
	mm.logger = logger
}

func (mm *moduleManager) InitAndUpdateModule(ctx context.Context, dir string) (Module, error) {
	mod, err := mm.ModuleByPath(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get module: %+v", err)
	}

	if err := mod.ExecuteTerraformInit(ctx); err != nil {
		return nil, fmt.Errorf("failed to init module: %+v", err)
	}

	m := mod.(*module)
	m.discoverCaches(ctx, dir)
	return mod, m.UpdateProviderSchemaCache(ctx, m.pluginLockFile)
}

func (mm *moduleManager) AddAndStartLoadingModule(ctx context.Context, dir string) (Module, error) {
	dir = filepath.Clean(dir)

	// TODO: Follow symlinks (requires proper test data)

	if _, ok := mm.moduleByPath(dir); ok {
		return nil, fmt.Errorf("module %s was already added", dir)
	}

	mod, err := mm.newModule(context.Background(), dir)
	if err != nil {
		return nil, err
	}

	mm.modules = append(mm.modules, mod)

	if mm.syncLoading {
		mm.logger.Printf("synchronously loading module %s", dir)
		return mod, mod.load(ctx)
	}

	mm.logger.Printf("asynchronously loading module %s", dir)
	mm.workerPool.Submit(func() {
		mod := mod
		err := mod.load(context.Background())
		mod.setLoadErr(err)
	})

	return mod, nil
}

func (mm *moduleManager) SchemaForPath(path string) (*schema.BodySchema, error) {
	candidates := mm.ModuleCandidatesByPath(path)
	for _, mod := range candidates {
		schema, err := mod.MergedSchema()
		if err != nil {
			mm.logger.Printf("failed to merge schema for %s: %s", mod.Path(), err)
			continue
		}
		if schema != nil {
			mm.logger.Printf("found schema for %s at %s", path, mod.Path())
			return schema, nil
		}
	}

	mod, err := mm.ModuleByPath(path)
	if err != nil {
		return nil, err
	}

	return mod.MergedSchema()
}

func (mm *moduleManager) moduleByPath(dir string) (*module, bool) {
	for _, mod := range mm.modules {
		if pathEquals(mod.Path(), dir) {
			return mod, true
		}
	}
	return nil, false
}

// ModuleCandidatesByPath finds any initialized modules
func (mm *moduleManager) ModuleCandidatesByPath(path string) Modules {
	path = filepath.Clean(path)

	candidates := make([]Module, 0)

	// TODO: Follow symlinks (requires proper test data)

	mod, foundPath := mm.moduleByPath(path)
	if foundPath {
		inited, _ := mod.WasInitialized()
		if inited {
			candidates = append(candidates, mod)
		}
	}

	if !foundPath {
		dir := trimLockFilePath(path)
		mod, ok := mm.moduleByPath(dir)
		if ok {
			inited, _ := mod.WasInitialized()
			if inited {
				candidates = append(candidates, mod)
			}
		}
	}

	for _, mod := range mm.modules {
		if mod.ReferencesModulePath(path) {
			candidates = append(candidates, mod)
		}
	}

	return candidates
}

func (mm *moduleManager) ListModules() Modules {
	modules := make([]Module, 0)
	for _, mod := range mm.modules {
		modules = append(modules, mod)
	}
	return modules
}
func (mm *moduleManager) ModuleByPath(path string) (Module, error) {
	path = filepath.Clean(path)

	if mod, ok := mm.moduleByPath(path); ok {
		return mod, nil
	}

	dir := trimLockFilePath(path)

	if mod, ok := mm.moduleByPath(dir); ok {
		return mod, nil
	}

	return nil, &ModuleNotFoundErr{path}
}

func (mm *moduleManager) IsProviderSchemaLoaded(path string) (bool, error) {
	mod, err := mm.ModuleByPath(path)
	if err != nil {
		return false, err
	}

	return mod.IsProviderSchemaLoaded(), nil
}

func (mm *moduleManager) TerraformFormatterForDir(ctx context.Context, path string) (exec.Formatter, error) {
	mod, err := mm.ModuleByPath(path)
	if err != nil {
		if IsModuleNotFound(err) {
			return mm.newTerraformFormatter(ctx, path)
		}
		return nil, err
	}

	return mod.TerraformFormatter()
}

func (mm *moduleManager) newTerraformFormatter(ctx context.Context, workDir string) (exec.Formatter, error) {
	tfPath := mm.tfExecPath
	if tfPath == "" {
		var err error
		tfPath, err = mm.tfDiscoFunc()
		if err != nil {
			return nil, err
		}
	}

	tf, err := mm.tfNewExecutor(workDir, tfPath)
	if err != nil {
		return nil, err
	}

	tf.SetLogger(mm.logger)

	if mm.tfExecLogPath != "" {
		tf.SetExecLogPath(mm.tfExecLogPath)
	}

	if mm.tfExecTimeout != 0 {
		tf.SetTimeout(mm.tfExecTimeout)
	}

	version, _, err := tf.Version(ctx)
	if err != nil {
		return nil, err
	}
	mm.logger.Printf("Terraform version %s found at %s (alternative)", version, tf.GetExecPath())

	return tf.Format, nil
}

func (mm *moduleManager) IsTerraformAvailable(path string) (bool, error) {
	mod, err := mm.ModuleByPath(path)
	if err != nil {
		return false, err
	}

	return mod.IsTerraformAvailable(), nil
}

func (mm *moduleManager) HasTerraformDiscoveryFinished(path string) (bool, error) {
	mod, err := mm.ModuleByPath(path)
	if err != nil {
		return false, err
	}

	return mod.HasTerraformDiscoveryFinished(), nil
}

func (mm *moduleManager) CancelLoading() {
	for _, mod := range mm.modules {
		mm.logger.Printf("cancelling loading for %s", mod.Path())
		mod.CancelLoading()
		mm.logger.Printf("loading cancelled for %s", mod.Path())
	}
	mm.workerPool.Stop()
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

func (mm *moduleManager) PathsToWatch() []string {
	paths := make([]string, 0)
	for _, mod := range mm.modules {
		ptw := mod.PathsToWatch()
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
func NewModuleLoader(ctx context.Context, mm ModuleManager) ModuleLoader {
	return func(dir string) (Module, error) {
		return mm.AddAndStartLoadingModule(ctx, dir)
	}
}
