package rootmodule

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
	logger  *log.Logger
	sync    bool
	walking bool

	cancelFunc context.CancelFunc
	doneCh     chan struct{}
}

func NewWalker() *Walker {
	return &Walker{
		logger: discardLogger,
		doneCh: make(chan struct{}, 0),
	}
}

func (w *Walker) SetLogger(logger *log.Logger) {
	w.logger = logger
}

type WalkFunc func(ctx context.Context, rootModulePath string) error

func (w *Walker) Stop() {
	if w.cancelFunc != nil {
		w.cancelFunc()
	}

	if w.walking {
		w.walking = false
		w.doneCh <- struct{}{}
	}
}

func (w *Walker) StartWalking(path string, wf WalkFunc) error {
	if w.walking {
		return fmt.Errorf("walker is already running")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	w.cancelFunc = cancelFunc
	w.walking = true

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
	return w.walking
}

func (w *Walker) walk(ctx context.Context, rootPath string, wf WalkFunc) error {
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

		if info.Name() == ".terraform" {
			rootDir, err := filepath.Abs(filepath.Dir(path))
			if err != nil {
				return err
			}

			w.logger.Printf("found root module %s", rootDir)
			return wf(ctx, rootDir)
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
	w.walking = false
	return err
}

func isSkippableDir(dirName string) bool {
	_, ok := skipDirNames[dirName]
	return ok
}
