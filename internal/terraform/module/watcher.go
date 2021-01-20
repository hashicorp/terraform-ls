package module

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
)

// Watcher is a wrapper around native fsnotify.Watcher
// It provides the ability to detect actual file changes
// (rather than just events that may not be changing any bytes)
type watcher struct {
	fw      *fsnotify.Watcher
	fs      filesystem.Filesystem
	modMgr  ModuleManager
	modules []*watchedModule
	logger  *log.Logger

	watching   bool
	cancelFunc context.CancelFunc
}

type WatcherFactory func(filesystem.Filesystem, ModuleManager) (Watcher, error)

type watchedModule struct {
	Path      string
	Watched   []string
	Watchable *datadir.WatchablePaths
}

func NewWatcher(fs filesystem.Filesystem, modMgr ModuleManager) (Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &watcher{
		fw:      fw,
		fs:      fs,
		modMgr:  modMgr,
		logger:  defaultLogger,
		modules: make([]*watchedModule, 0),
	}, nil
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

func (w *watcher) SetLogger(logger *log.Logger) {
	w.logger = logger
}

func (w *watcher) IsModuleWatched(modPath string) bool {
	modPath = filepath.Clean(modPath)

	for _, m := range w.modules {
		if pathEquals(m.Path, modPath) {
			return true
		}
	}

	return false
}

func (w *watcher) AddModule(modPath string) error {
	modPath = filepath.Clean(modPath)

	w.logger.Printf("adding module for watching: %s", modPath)

	wm := &watchedModule{
		Path:      modPath,
		Watched:   make([]string, 0),
		Watchable: datadir.WatchableModulePaths(modPath),
	}
	w.modules = append(w.modules, wm)

	// We watch individual dirs (instead of individual files).
	// This does result in more events but fewer watched paths.
	// fsnotify does not support recursive watching yet.
	// See https://github.com/fsnotify/fsnotify/issues/18

	err := w.fw.Add(modPath)
	if err != nil {
		return err
	}

	for _, dirPath := range wm.Watchable.Dirs {
		err := w.fw.Add(dirPath)
		if err == nil {
			wm.Watched = append(wm.Watched, dirPath)
		}
	}

	return nil
}

func (w *watcher) run(ctx context.Context) {
	for {
		select {
		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}
			w.processEvent(event)
		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			w.logger.Println("watch error:", err)
		}
	}
}

func (w *watcher) processEvent(event fsnotify.Event) {
	eventPath := event.Name

	if event.Op&fsnotify.Write == fsnotify.Write {
		for _, mod := range w.modules {
			if containsPath(mod.Watchable.ModuleManifests, eventPath) {
				w.modMgr.EnqueueModuleOp(mod.Path, OpTypeParseModuleManifest)
				return
			}
			if containsPath(mod.Watchable.PluginLockFiles, eventPath) {
				w.modMgr.EnqueueModuleOp(mod.Path, OpTypeObtainSchema)
				return
			}
		}
	}

	if event.Op&fsnotify.Create == fsnotify.Create {
		for _, mod := range w.modules {
			if containsPath(mod.Watchable.Dirs, eventPath) {
				w.fw.Add(eventPath)
				mod.Watched = append(mod.Watched, eventPath)

				filepath.Walk(eventPath, func(path string, info os.FileInfo, err error) error {
					if info.IsDir() {
						if containsPath(mod.Watchable.Dirs, path) {
							w.fw.Add(path)
							mod.Watched = append(mod.Watched, path)
						}
						return nil
					}
					if containsPath(mod.Watchable.ModuleManifests, path) {
						return w.modMgr.EnqueueModuleOp(mod.Path, OpTypeParseModuleManifest)
					}
					if containsPath(mod.Watchable.PluginLockFiles, path) {
						return w.modMgr.EnqueueModuleOp(mod.Path, OpTypeObtainSchema)
					}
					return nil
				})

				return
			}

			if containsPath(mod.Watchable.ModuleManifests, eventPath) {
				w.modMgr.EnqueueModuleOp(mod.Path, OpTypeParseModuleManifest)
				return
			}

			if containsPath(mod.Watchable.PluginLockFiles, eventPath) {
				w.modMgr.EnqueueModuleOp(mod.Path, OpTypeObtainSchema)
				return
			}
		}
	}

	if event.Op&fsnotify.Remove == fsnotify.Remove {
		for modI, mod := range w.modules {
			// Whole module being removed
			if pathEquals(mod.Path, eventPath) {
				for _, wPath := range mod.Watched {
					w.fw.Remove(wPath)
				}
				w.fw.Remove(mod.Path)
				w.modules = append(w.modules[:modI], w.modules[modI+1:]...)
				return
			}

			for i, wp := range mod.Watched {
				if pathEquals(wp, eventPath) {
					w.fw.Remove(wp)
					mod.Watched = append(mod.Watched[:i], mod.Watched[i+1:]...)
					return
				}
			}
		}
	}
}

func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if pathEquals(p, path) {
			return true
		}
	}
	return false
}

func (w *watcher) Start() error {
	if w.watching {
		w.logger.Println("watching already in progress")
		return nil
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	w.cancelFunc = cancelFunc
	w.watching = true

	w.logger.Printf("watching for changes ...")
	go w.run(ctx)

	return nil
}

func (w *watcher) Stop() error {
	if !w.watching {
		return nil
	}

	w.cancelFunc()

	err := w.fw.Close()
	if err == nil {
		w.watching = false
	}

	return err
}
