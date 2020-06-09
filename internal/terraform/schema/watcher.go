package schema

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/terraform-ls/internal/terraform/errors"
)

type watcher interface {
	AddWorkspace(string) error
	Close() error
	Events() chan fsnotify.Event
	Errors() chan error
	OnPluginChange(func(*watchedWorkspace) error)
	SetLogger(*log.Logger)
}

func lockFilePaths(dir string) []string {
	return []string{
		// Terraform >= v0.13
		filepath.Join(dir,
			".terraform",
			"plugins",
			"selections.json"),
		// Terraform <= v0.12
		filepath.Join(dir,
			".terraform",
			"plugins",
			runtime.GOOS+"_"+runtime.GOARCH,
			"lock.json"),
	}
}

// Watcher is a wrapper around native fsnotify.Watcher
// to make it swappable for MockWatcher via interface,
// provide higher-level ability to detect actual file changes
// (rather than just events that may not be changing any bytes)
// and hold knowledge about workspace structure
type Watcher struct {
	w         *fsnotify.Watcher
	lockFiles map[string]*watchedWorkspace
	logger    *log.Logger
}

type watchedWorkspace struct {
	pluginsLockFileHash string
	dir                 string
}

func NewWatcher() (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		w:         w,
		lockFiles: make(map[string]*watchedWorkspace, 0),
		logger:    defaultLogger,
	}, nil
}

func (w *Watcher) SetLogger(logger *log.Logger) {
	w.logger = logger
}

func (w *Watcher) AddWorkspace(dir string) error {
	lockPaths := lockFilePaths(dir)
	w.logger.Printf("Adding %q for watching...", lockPaths)

	lockFile, err := findLockFile(lockPaths)
	if err != nil {
		if os.IsNotExist(err) {
			return &errors.NotInitializedErr{
				Dir: dir,
			}
		}
		return fmt.Errorf("unable to calculate hash: %w", err)
	}

	w.lockFiles[lockFile.path] = &watchedWorkspace{
		pluginsLockFileHash: lockFile.hash,
		dir:                 dir,
	}

	return w.w.Add(lockFile.path)
}

type lockFile struct {
	path string
	hash string
}

func findLockFile(paths []string) (*lockFile, error) {
	var b []byte
	var err error

	for _, path := range paths {
		b, err = fileHashSum(path)
		if err == nil {
			return &lockFile{
				path: path,
				hash: string(b),
			}, nil
		}

		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return nil, err
}

func fileHashSum(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func (w *Watcher) Close() error {
	return w.w.Close()
}

func (w *Watcher) Events() chan fsnotify.Event {
	return w.w.Events
}

func (w *Watcher) Errors() chan error {
	return w.w.Errors
}

func (w *Watcher) OnPluginChange(f func(*watchedWorkspace) error) {
	for {
		select {
		case event, ok := <-w.Events():
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				hash, err := fileHashSum(event.Name)
				if err != nil {
					w.logger.Println("unable to calculate hash:", err)
				}
				newHash := string(hash)
				existingHash := w.lockFiles[event.Name].pluginsLockFileHash

				if newHash != existingHash {
					w.lockFiles[event.Name].pluginsLockFileHash = newHash

					err = f(w.lockFiles[event.Name])
					if err != nil {
						w.logger.Println("error when executing on change:", err)
					}
				}
			}
		case err, ok := <-w.Errors():
			if !ok {
				return
			}
			w.logger.Println("watch error:", err)
		}
	}
}
