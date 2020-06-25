package watcher

import (
	"context"
	"io/ioutil"
	"log"

	"github.com/fsnotify/fsnotify"
)

// Watcher is a wrapper around native fsnotify.Watcher
// It provides the ability to detect actual file changes
// (rather than just events that may not be changing any bytes)
type watcher struct {
	fw           *fsnotify.Watcher
	trackedFiles map[string]TrackedFile
	changeHooks  []ChangeHook
	logger       *log.Logger

	watching   bool
	cancelFunc context.CancelFunc
}

type WatcherFactory func() (Watcher, error)

func NewWatcher() (Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &watcher{
		fw:           fw,
		logger:       defaultLogger,
		trackedFiles: make(map[string]TrackedFile, 0),
	}, nil
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

func (w *watcher) SetLogger(logger *log.Logger) {
	w.logger = logger
}

func (w *watcher) AddPaths(paths []string) error {
	for _, p := range paths {
		err := w.AddPath(p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *watcher) AddPath(path string) error {
	w.logger.Printf("adding %s for watching", path)

	tf, err := trackedFileFromPath(path)
	if err != nil {
		return err
	}
	w.trackedFiles[path] = tf

	return w.fw.Add(path)
}

func (w *watcher) AddChangeHook(h ChangeHook) {
	w.changeHooks = append(w.changeHooks, h)
}

func (w *watcher) run(ctx context.Context) {
	for {
		select {
		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				w.logger.Printf("detected write into %s", event.Name)
				oldTf := w.trackedFiles[event.Name]
				newTf, err := trackedFileFromPath(event.Name)
				if err != nil {
					w.logger.Println("failed to track file, ignoring", err)
					continue
				}
				w.trackedFiles[event.Name] = newTf

				if oldTf.Sha256Sum() != newTf.Sha256Sum() {
					for _, h := range w.changeHooks {
						err := h(ctx, newTf)
						if err != nil {
							w.logger.Println("change hook error:", err)
						}
					}
				}
			}
		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			w.logger.Println("watch error:", err)
		}
	}
}

// StartWatching starts to watch for changes that were added
// via AddPath(s) until Stop() is called
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
