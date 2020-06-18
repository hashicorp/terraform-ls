package watcher

import (
	"log"
)

func MockWatcher() WatcherFactory {
	return func() (Watcher, error) {
		return &mockWatcher{}, nil
	}
}

type mockWatcher struct{}

func (w *mockWatcher) AddChangeHook(h ChangeHook) {
}

func (w *mockWatcher) AddPaths(paths []string) error {
	return nil
}

func (w *mockWatcher) AddPath(path string) error {
	return nil
}

func (w *mockWatcher) Start() error {
	return nil
}

func (w *mockWatcher) Stop() error {
	return nil
}

func (w *mockWatcher) SetLogger(*log.Logger) {}
