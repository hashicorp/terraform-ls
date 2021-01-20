package module

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
)

var (
	discardLogger = log.New(ioutil.Discard, "", 0)

	// skipDirNames represent directory names which would never contain
	// plugin/module cache, so it's safe to skip them during the walk
	skipDirNames = map[string]bool{
		".git":                true,
		".idea":               true,
		".vscode":             true,
		"terraform.tfstate.d": true,
	}
)

type Walker struct {
	fs      filesystem.Filesystem
	modMgr  ModuleManager
	watcher Watcher
	logger  *log.Logger
	sync    bool

	walking    bool
	walkingMu  *sync.RWMutex
	cancelFunc context.CancelFunc
	doneCh     <-chan struct{}

	excludeModulePaths map[string]bool
}

func NewWalker(fs filesystem.Filesystem, modMgr ModuleManager) *Walker {
	return &Walker{
		fs:        fs,
		modMgr:    modMgr,
		logger:    discardLogger,
		walkingMu: &sync.RWMutex{},
		doneCh:    make(chan struct{}, 0),
	}
}

func (w *Walker) SetLogger(logger *log.Logger) {
	w.logger = logger
}

func (w *Walker) SetWatcher(watcher Watcher) {
	w.watcher = watcher
}

func (w *Walker) SetExcludeModulePaths(excludeModulePaths []string) {
	w.excludeModulePaths = make(map[string]bool)
	for _, path := range excludeModulePaths {
		w.excludeModulePaths[path] = true
	}
}

func (w *Walker) Stop() {
	if w.cancelFunc != nil {
		w.cancelFunc()
	}

	if w.IsWalking() {
		w.logger.Println("stopping walker")
		w.setWalking(false)
	}
}

func (w *Walker) setWalking(isWalking bool) {
	w.walkingMu.Lock()
	defer w.walkingMu.Unlock()
	w.walking = isWalking
}

func (w *Walker) StartWalking(ctx context.Context, path string) error {
	if w.IsWalking() {
		return fmt.Errorf("walker is already running")
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	w.cancelFunc = cancelFunc
	w.doneCh = ctx.Done()
	w.setWalking(true)

	if w.sync {
		w.logger.Printf("synchronously walking through %s", path)
		return w.walk(ctx, path)
	}

	go func(w *Walker, path string) {
		w.logger.Printf("asynchronously walking through %s", path)
		err := w.walk(ctx, path)
		if err != nil {
			w.logger.Printf("async walking through %s failed: %s", path, err)
			return
		}
		w.logger.Printf("async walking through %s finished", path)
	}(w, path)

	return nil
}

func (w *Walker) IsWalking() bool {
	w.walkingMu.RLock()
	defer w.walkingMu.RUnlock()

	return w.walking
}

func (w *Walker) walk(ctx context.Context, rootPath string) error {
	defer w.Stop()

	// We ignore the passed FS and instead read straight from OS FS
	// because that would require reimplementing filepath.Walk and
	// the data directory should never be on the virtual filesystem anyway
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		select {
		case <-w.doneCh:
			w.logger.Printf("cancelling walk of %s...", rootPath)
			return fmt.Errorf("walk cancelled")
		default:
		}

		if err != nil {
			w.logger.Printf("unable to access %s: %s", path, err.Error())
			return nil
		}

		dir, err := filepath.Abs(filepath.Dir(path))
		if err != nil {
			return err
		}

		if _, ok := w.excludeModulePaths[dir]; ok {
			return filepath.SkipDir
		}

		if info.Name() == datadir.DataDirName {
			w.logger.Printf("found module %s", dir)

			_, err := w.modMgr.ModuleByPath(dir)
			if err != nil {
				if IsModuleNotFound(err) {
					_, err := w.modMgr.AddModule(dir)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}

			err = w.modMgr.EnqueueModuleOp(dir, OpTypeGetTerraformVersion)
			if err != nil {
				return err
			}

			dataDir := datadir.WalkDataDirOfModule(w.fs, dir)
			if dataDir.ModuleManifestPath != "" {
				err = w.modMgr.EnqueueModuleOp(dir, OpTypeParseModuleManifest)
				if err != nil {
					return err
				}
			}
			if dataDir.PluginLockFilePath != "" {
				err = w.modMgr.EnqueueModuleOp(dir, OpTypeObtainSchema)
				if err != nil {
					return err
				}
			}

			if w.watcher != nil {
				w.watcher.AddModule(dir)
			}

			return nil
		}

		if !info.IsDir() {
			// All files are skipped, we only care about dirs
			return nil
		}

		if isSkippableDir(info.Name()) {
			w.logger.Printf("skipping %s", path)
			return filepath.SkipDir
		}

		return nil
	})
	w.logger.Printf("walking of %s finished", rootPath)
	return err
}

func isSkippableDir(dirName string) bool {
	_, ok := skipDirNames[dirName]
	return ok
}
