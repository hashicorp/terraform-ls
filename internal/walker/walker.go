package walker

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/terraform/ast"
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
	fs        fs.ReadDirFS
	pathStore PathStore
	modStore  ModuleStore

	logger   *log.Logger
	walkFunc WalkFunc

	Collector *WalkerCollector

	cancelFunc context.CancelFunc

	ignoredPaths          map[string]bool
	ignoredDirectoryNames map[string]bool
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

func NewWalker(fs fs.ReadDirFS, pathStore PathStore, modStore ModuleStore, walkFunc WalkFunc) *Walker {
	return &Walker{
		fs:                    fs,
		pathStore:             pathStore,
		modStore:              modStore,
		walkFunc:              walkFunc,
		logger:                discardLogger,
		ignoredDirectoryNames: skipDirNames,
	}
}

func (w *Walker) SetLogger(logger *log.Logger) {
	w.logger = logger
}

func (w *Walker) SetIgnoredPaths(ignoredPaths []string) {
	w.ignoredPaths = make(map[string]bool)
	for _, path := range ignoredPaths {
		w.ignoredPaths[path] = true
	}
}

func (w *Walker) SetIgnoredDirectoryNames(ignoredDirectoryNames []string) {
	if w.cancelFunc != nil {
		panic("cannot set ignorelist after walking started")
	}
	for _, path := range ignoredDirectoryNames {
		w.ignoredDirectoryNames[path] = true
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
	_, ok := w.ignoredDirectoryNames[dirName]
	return ok
}

func (w *Walker) walk(ctx context.Context, dir document.DirHandle) error {
	if _, ok := w.ignoredPaths[dir.Path()]; ok {
		w.logger.Printf("skipping walk due to dir being excluded: %s", dir.Path())
		return nil
	}

	dirEntries, err := fs.ReadDir(w.fs, dir.Path())
	if err != nil {
		w.logger.Printf("reading directory failed: %s: %s", dir.Path(), err)
		// fs.ReadDir (or at least the os.ReadDir implementation) returns
		// the entries it was able to read before the error, along with the error.
	}

	dirIndexed := false

	for _, dirEntry := range dirEntries {
		select {
		case <-ctx.Done():
			w.logger.Printf("cancelling walk of %s...", dir)
			return fmt.Errorf("walk cancelled")
		default:
		}

		if w.isSkippableDir(dirEntry.Name()) {
			w.logger.Printf("skipping ignored dir name: %s", dirEntry.Name())
			continue
		}

		if !dirIndexed && ast.IsModuleFilename(dirEntry.Name()) && !ast.IsIgnoredFile(dirEntry.Name()) {
			dirIndexed = true
			w.logger.Printf("found module %s", dir)

			exists, err := w.modStore.Exists(dir.Path())
			if err != nil {
				return err
			}
			if !exists {
				err := w.modStore.Add(dir.Path())
				if err != nil {
					return err
				}
			}

			ids, err := w.walkFunc(ctx, dir)
			if err != nil {
				w.collectError(err)
			}
			w.collectJobIds(ids)
			continue
		}

		if dirEntry.IsDir() {
			path := filepath.Join(dir.Path(), dirEntry.Name())
			dirHandle := document.DirHandleFromPath(path)
			err = w.walk(ctx, dirHandle)
			if err != nil {
				return err
			}
		}
	}
	w.logger.Printf("walking of %s finished", dir)
	return err
}
