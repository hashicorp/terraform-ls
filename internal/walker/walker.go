package walker

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
)

var (
	discardLogger = log.New(ioutil.Discard, "", 0)

	// skipDirNames represent directory names which would never contain
	// plugin/module cache, so it's safe to skip them during the walk
	//
	// please keep the list in `SETTINGS.md` in sync
	skipDirNames = map[string]bool{
		".git":                true,
		".idea":               true,
		".vscode":             true,
		"terraform.tfstate.d": true,
		".terragrunt-cache":   true,
	}
)

type pathToWatch struct{}

type Walker struct {
	pathStore PathStore
	modStore  ModuleStore

	logger   *log.Logger
	walkFunc WalkFunc

	Collector *WalkerCollector

	cancelFunc context.CancelFunc

	excludeModulePaths   map[string]bool
	ignoreDirectoryNames map[string]bool
}

type WalkFunc func(ctx context.Context, modHandle document.DirHandle) (job.IDs, error)

type PathStore interface {
	AwaitNextDir(ctx context.Context) (document.DirHandle, error)
	RemoveDir(dir document.DirHandle) error
}

type ModuleStore interface {
	Exists(dir string) (bool, error)
	Add(dir string) error
}

func NewWalker(pathStore PathStore, modStore ModuleStore, walkFunc WalkFunc) *Walker {
	return &Walker{
		pathStore:            pathStore,
		modStore:             modStore,
		walkFunc:             walkFunc,
		logger:               discardLogger,
		ignoreDirectoryNames: skipDirNames,
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

func (w *Walker) SetIgnoreDirectoryNames(ignoreDirectoryNames []string) {
	if w.cancelFunc != nil {
		panic("cannot set ignorelist after walking started")
	}
	for _, path := range ignoreDirectoryNames {
		w.ignoreDirectoryNames[path] = true
	}
}

func (w *Walker) Stop() {
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
}

func (w *Walker) StartWalking(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	w.cancelFunc = cancelFunc

	go func() {
		for {
			nextDir, err := w.pathStore.AwaitNextDir(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				w.logger.Printf("walker: awaiting next dir failed: %s", err)
				w.collectError(err)
				return
			}

			err = w.walk(ctx, nextDir)
			if err != nil {
				w.logger.Printf("walker: walking through %q failed: %s", nextDir, err)
				w.collectError(err)
				continue
			}

			err = w.pathStore.RemoveDir(nextDir)
			if err != nil {
				w.logger.Printf("walker: removing dir %q from queue failed: %s", nextDir, err)
				w.collectError(err)
				continue
			}
			w.logger.Printf("walker: walking through %q finished", nextDir)

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return nil
}

func (w *Walker) collectError(err error) {
	if w.Collector != nil {
		w.Collector.CollectError(err)
	}
}

func (w *Walker) collectJobIds(jobIds job.IDs) {
	if w.Collector != nil {
		for _, id := range jobIds {
			w.Collector.CollectJobId(id)
		}
	}
}

func (w *Walker) isSkippableDir(dirName string) bool {
	_, ok := w.ignoreDirectoryNames[dirName]
	return ok
}

func (w *Walker) walk(ctx context.Context, dir document.DirHandle) error {
	// We ignore the passed FS and instead read straight from OS FS
	// because that would require reimplementing filepath.Walk and
	// the data directory should never be on the virtual filesystem anyway
	err := filepath.Walk(dir.Path(), func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			w.logger.Printf("cancelling walk of %s...", dir)
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

		if w.isSkippableDir(info.Name()) {
			w.logger.Printf("skipping %s", path)
			return filepath.SkipDir
		}

		if _, ok := w.excludeModulePaths[dir]; ok {
			return filepath.SkipDir
		}

		if info.Name() == datadir.DataDirName {
			w.logger.Printf("found module %s", dir)

			_, err := w.modStore.Exists(dir)
			if err != nil {
				if state.IsModuleNotFound(err) {
					err := w.modStore.Add(dir)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}

			modHandle := document.DirHandleFromPath(dir)
			ids, err := w.walkFunc(ctx, modHandle)
			if err != nil {
				w.collectError(err)
			}
			w.collectJobIds(ids)

			return nil
		}

		if !info.IsDir() {
			// All files are skipped, we only care about dirs
			return nil
		}

		return nil
	})
	w.logger.Printf("walking of %s finished", dir)
	return err
}
