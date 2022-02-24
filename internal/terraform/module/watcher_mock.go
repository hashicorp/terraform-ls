package module

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

func MockWatcher() WatcherFactory {
	return func(fs ReadOnlyFS, ms *state.ModuleStore, pss *state.ProviderSchemaStore, js job.JobStore, tfExec exec.ExecutorFactory) (Watcher, error) {
		return &mockWatcher{}, nil
	}
}

type mockWatcher struct{}

func (w *mockWatcher) Start(context.Context) error {
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
