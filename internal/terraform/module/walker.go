package module

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
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
	logger *log.Logger
	sync   bool

	walking    bool
	walkingMu  *sync.RWMutex
	cancelFunc context.CancelFunc
	doneCh     <-chan struct{}

	excludeModulePaths map[string]bool
}

func NewWalker() *Walker {
	return &Walker{
		logger:    discardLogger,
		walkingMu: &sync.RWMutex{},
		doneCh:    make(chan struct{}, 0),
	}
}

func (w *Walker) SetLogger(logger *log.Logger) {
	w.logger = logger
}

func (w *Walker) SetExcludeModulePaths(excludeModulePaths []string) {
	w.excludeModulePaths = make(map[string]bool)
	for _, path := range excludeModulePaths {
		w.excludeModulePaths[path] = true
	}
}

type WalkFunc func(ctx context.Context, rootModulePath string) error

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

func (w *Walker) Done() <-chan struct{} {
	return w.doneCh
}

func (w *Walker) StartWalking(ctx context.Context, path string, wf WalkFunc) error {
	if w.IsWalking() {
		return fmt.Errorf("walker is already running")
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	w.cancelFunc = cancelFunc
	w.doneCh = ctx.Done()
	w.setWalking(true)

	if w.sync {
		w.logger.Printf("synchronously walking through %s", path)
		return w.walk(ctx, path, wf)
	}

	go func(w *Walker, path string, wf WalkFunc) {
		w.logger.Printf("asynchronously walking through %s", path)
		err := w.walk(ctx, path, wf)
		if err != nil {
			w.logger.Printf("async walking through %s failed: %s", path, err)
			return
		}
		w.logger.Printf("async walking through %s finished", path)
	}(w, path, wf)

	return nil
}

func (w *Walker) IsWalking() bool {
	w.walkingMu.RLock()
	defer w.walkingMu.RUnlock()

	return w.walking
}

func (w *Walker) walk(ctx context.Context, rootPath string, wf WalkFunc) error {
	defer w.Stop()

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

		if info.Name() == ".terraform" {
			w.logger.Printf("found module %s", dir)
			return wf(ctx, dir)
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
