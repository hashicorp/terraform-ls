package rootmodule

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
)

type rootModule struct {
	ctx                context.Context
	logger             *log.Logger
	pluginLockFile     File
	moduleManifestFile File
	moduleManifest     *moduleManifest
	tfVersion          string

	tfDiscoFunc       discovery.DiscoveryFunc
	tfNewExecutor     exec.ExecutorFactory
	tfExecPath        string
	tfExecTimeout     time.Duration
	tfExecLogPath     string
	newSchemaStorage  schema.StorageFactory
	ignorePluginCache bool

	tfExec       *exec.Executor
	parser       lang.Parser
	schemaWriter schema.Writer
	pluginMu     *sync.RWMutex
	moduleMu     *sync.RWMutex
}

func newRootModule(ctx context.Context) *rootModule {
	return &rootModule{
		ctx:      ctx,
		logger:   defaultLogger,
		pluginMu: &sync.RWMutex{},
		moduleMu: &sync.RWMutex{},
	}
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

func NewRootModule(ctx context.Context, dir string) (RootModule, error) {
	rm := newRootModule(ctx)

	d := &discovery.Discovery{}
	rm.tfDiscoFunc = d.LookPath

	rm.tfNewExecutor = exec.NewExecutor
	rm.newSchemaStorage = func() *schema.Storage {
		ss := schema.NewStorage()
		ss.SetSynchronous()
		return ss
	}

	return rm, rm.init(ctx, dir)
}

func (rm *rootModule) SetLogger(logger *log.Logger) {
	rm.logger = logger
}

func (rm *rootModule) init(ctx context.Context, dir string) error {
	rm.logger.Printf("initing new root module: %s", dir)
	tf, err := rm.initTfExecutor(dir)
	if err != nil {
		return err
	}

	version, err := tf.Version()
	if err != nil {
		return err
	}

	rm.logger.Printf("Terraform version %s found at %s", version, tf.GetExecPath())

	err = schema.SchemaSupportsTerraform(version)
	if err != nil {
		return err
	}

	p, err := lang.FindCompatibleParser(version)
	if err != nil {
		return err
	}
	p.SetLogger(rm.logger)

	ss := rm.newSchemaStorage()

	ss.SetLogger(rm.logger)

	p.SetSchemaReader(ss)

	rm.parser = p
	rm.schemaWriter = ss
	rm.tfExec = tf
	rm.tfVersion = version

	err = rm.initPluginCache(dir)
	if err != nil {
		return fmt.Errorf("plugin initialization failed: %w", err)
	}
	err = rm.initModuleCache(dir)
	if err != nil {
		return err
	}
	return nil
}

func (rm *rootModule) initTfExecutor(dir string) (*exec.Executor, error) {
	tfPath := rm.tfExecPath
	if tfPath == "" {
		var err error
		tfPath, err = rm.tfDiscoFunc()
		if err != nil {
			return nil, err
		}
	}

	tf := rm.tfNewExecutor(rm.ctx, tfPath)

	tf.SetWorkdir(dir)
	tf.SetLogger(rm.logger)

	if rm.tfExecLogPath != "" {
		tf.SetExecLogPath(rm.tfExecLogPath)
	}

	if rm.tfExecTimeout != 0 {
		tf.SetTimeout(rm.tfExecTimeout)
	}

	return tf, nil
}

func (rm *rootModule) initPluginCache(dir string) error {
	var lf File
	if rm.ignorePluginCache {
		lf = &file{
			path: pluginLockFilePaths(dir)[0],
		}
	} else {
		var err error
		lockPaths := pluginLockFilePaths(dir)
		lf, err = findFile(lockPaths)
		if err != nil {
			if os.IsNotExist(err) {
				rm.logger.Printf("no plugin cache found: %s", err.Error())
				return nil
			}

			return fmt.Errorf("unable to calculate hash: %w", err)
		}
	}

	return rm.UpdatePluginCache(lf)
}

func findFile(paths []string) (File, error) {
	var lf File
	var err error

	for _, path := range paths {
		lf, err = newFile(path)
		if err == nil {
			return lf, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return nil, err
}

type file struct {
	path string
}

func (f *file) Path() string {
	return f.path
}

func newFile(path string) (File, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("expected %s to be a file, not a dir", path)
	}

	return &file{path: path}, nil
}

func (rm *rootModule) initModuleCache(dir string) error {
	lf, err := newFile(moduleManifestFilePath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			rm.logger.Printf("no module manifest file found: %s", err.Error())
			return nil
		}

		return fmt.Errorf("unable to calculate hash: %w", err)
	}

	return rm.UpdateModuleManifest(lf)
}

func (rm *rootModule) UpdateModuleManifest(lockFile File) error {
	rm.moduleMu.Lock()
	rm.logger.Printf("updating module manifest based on %s ...", lockFile.Path())
	defer rm.moduleMu.Unlock()

	rm.moduleManifestFile = lockFile

	mm, err := ParseModuleManifestFromFile(lockFile.Path())
	if err != nil {
		return err
	}

	rm.moduleManifest = mm
	rm.logger.Printf("updated module manifest - %d references parsed", len(mm.Records))
	return nil
}

func (rm *rootModule) Parser() lang.Parser {
	rm.pluginMu.RLock()
	defer rm.pluginMu.RUnlock()

	return rm.parser
}

func (rm *rootModule) ReferencesModulePath(path string) bool {
	if rm.moduleManifest == nil {
		return false
	}

	for _, m := range rm.moduleManifest.Records {
		if m.IsRoot() {
			// skip root module, as that's tracked separately
			continue
		}
		if m.IsExternal() {
			// skip external modules as these shouldn't be modified from cache
			continue
		}
		absPath := filepath.Join(rm.moduleManifest.rootDir, m.Dir)
		rm.logger.Printf("checking if %q == %q", absPath, path)
		if absPath == path {
			return true
		}
	}

	return false
}

func (rm *rootModule) TerraformExecutor() *exec.Executor {
	return rm.tfExec
}

func (rm *rootModule) UpdatePluginCache(lockFile File) error {
	rm.pluginMu.Lock()
	defer rm.pluginMu.Unlock()

	rm.pluginLockFile = lockFile

	return rm.schemaWriter.ObtainSchemasForModule(
		rm.tfExec, rootModuleDirFromFilePath(lockFile.Path()))
}

func (rm *rootModule) PathsToWatch() []string {
	rm.pluginMu.RLock()
	rm.moduleMu.RLock()
	defer rm.moduleMu.RUnlock()
	defer rm.pluginMu.RUnlock()

	files := make([]string, 0)
	if rm.pluginLockFile != nil {
		files = append(files, rm.pluginLockFile.Path())
	}
	if rm.moduleManifestFile != nil {
		files = append(files, rm.moduleManifestFile.Path())
	}

	return files
}

func (rm *rootModule) IsKnownModuleManifestFile(path string) bool {
	rm.pluginMu.RLock()
	defer rm.pluginMu.RUnlock()

	if rm.pluginLockFile == nil {
		return false
	}

	return rm.pluginLockFile.Path() == path
}

func (rm *rootModule) IsKnownPluginLockFile(path string) bool {
	rm.moduleMu.RLock()
	defer rm.moduleMu.RUnlock()

	if rm.moduleManifestFile == nil {
		return false
	}

	return rm.moduleManifestFile.Path() == path
}
