package module

import (
	"log"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
)

func MockWatcher() WatcherFactory {
	return func(filesystem.Filesystem, ModuleManager) (Watcher, error) {
		return &mockWatcher{}, nil
	}
}

type mockWatcher struct{}

func (w *mockWatcher) Start() error {
	return nil
}
func (w *mockWatcher) Stop() error {
	return nil
}

func (w *mockWatcher) SetLogger(*log.Logger) {}

func (w *mockWatcher) AddModule(string) error {
	return nil
}

func (w *mockWatcher) RemoveModule(string) error {
	return nil
}

func (w *mockWatcher) IsModuleWatched(string) bool {
	return false
}
