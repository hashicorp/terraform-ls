package module

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/pathcmp"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

// Watcher is a wrapper around native fsnotify.Watcher
// It provides the ability to detect actual file changes
// (rather than just events that may not be changing any bytes)
type watcher struct {
	fw *fsnotify.Watcher

	fs            ReadOnlyFS
	modStore      *state.ModuleStore
	schemaStore   *state.ProviderSchemaStore
	jobStore      job.JobStore
	tfExecFactory exec.ExecutorFactory

	modules []*watchedModule
	logger  *log.Logger

	watching   bool
	cancelFunc context.CancelFunc
}

type WatcherFactory func(fs ReadOnlyFS, ms *state.ModuleStore, pss *state.ProviderSchemaStore, js job.JobStore, tfExec exec.ExecutorFactory) (Watcher, error)

type watchedModule struct {
	Path      string
	Watched   []string
	Watchable *datadir.WatchablePaths
}

func NewWatcher(fs ReadOnlyFS, ms *state.ModuleStore, pss *state.ProviderSchemaStore, js job.JobStore, tfExec exec.ExecutorFactory) (Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &watcher{
		fw:            fw,
		fs:            fs,
		modStore:      ms,
		schemaStore:   pss,
		jobStore:      js,
		tfExecFactory: tfExec,
		logger:        defaultLogger,
		modules:       make([]*watchedModule, 0),
	}, nil
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

func (w *watcher) SetLogger(logger *log.Logger) {
	w.logger = logger
}

func (w *watcher) IsModuleWatched(modPath string) bool {
	modPath = filepath.Clean(modPath)

	for _, m := range w.modules {
		if pathcmp.PathEquals(m.Path, modPath) {
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

func (w *watcher) RemoveModule(modPath string) error {
	modPath = filepath.Clean(modPath)

	w.logger.Printf("removing module from watching: %s", modPath)

	for modI, mod := range w.modules {
		if pathcmp.PathEquals(mod.Path, modPath) {
			for _, wPath := range mod.Watched {
				w.fw.Remove(wPath)
			}
			w.fw.Remove(mod.Path)
			w.modules = append(w.modules[:modI], w.modules[modI+1:]...)
		}

		for i, wp := range mod.Watched {
			if pathcmp.PathEquals(wp, modPath) {
				w.fw.Remove(wp)
				mod.Watched = append(mod.Watched[:i], mod.Watched[i+1:]...)
			}
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
			w.processEvent(ctx, event)
		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			w.logger.Println("watch error:", err)
		}
	}
}

func (w *watcher) processEvent(ctx context.Context, event fsnotify.Event) {
	eventPath := event.Name

	if event.Op&fsnotify.Write == fsnotify.Write {
		for _, mod := range w.modules {
			modHandle := document.DirHandleFromPath(mod.Path)
			if containsPath(mod.Watchable.ModuleManifests, eventPath) {
				id, err := w.jobStore.EnqueueJob(job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						return ParseModuleManifest(w.fs, w.modStore, mod.Path)
					},
					Type:  op.OpTypeParseModuleManifest.String(),
					Defer: decodeCalledModulesFunc(w.fs, w.modStore, w.schemaStore, w, mod.Path),
				})
				if err == nil {
					w.jobStore.WaitForJobs(ctx, id)
					collectReferences(modHandle, w.modStore, w.schemaStore)(ctx, nil)
				}

				return
			}
			if containsPath(mod.Watchable.PluginLockFiles, eventPath) {
				w.jobStore.EnqueueJob(job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						ctx = exec.WithExecutorFactory(ctx, w.tfExecFactory)
						eo, ok := exec.ExecutorOptsFromContext(ctx)
						if ok {
							ctx = exec.WithExecutorOpts(ctx, eo)
						}

						return ObtainSchema(ctx, w.modStore, w.schemaStore, mod.Path)
					},
					Type: op.OpTypeObtainSchema.String(),
				})
				w.jobStore.EnqueueJob(job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						ctx = exec.WithExecutorFactory(ctx, w.tfExecFactory)
						eo, ok := exec.ExecutorOptsFromContext(ctx)
						if ok {
							ctx = exec.WithExecutorOpts(ctx, eo)
						}

						return GetTerraformVersion(ctx, w.modStore, mod.Path)
					},
					Type: op.OpTypeGetTerraformVersion.String(),
				})
				return
			}
		}
	}

	if event.Op&fsnotify.Create == fsnotify.Create {
		for _, mod := range w.modules {
			modHandle := document.DirHandleFromPath(mod.Path)

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

					modHandle := document.DirHandleFromPath(path)

					if containsPath(mod.Watchable.ModuleManifests, path) {
						id, err := w.jobStore.EnqueueJob(job.Job{
							Dir: modHandle,
							Func: func(ctx context.Context) error {
								return ParseModuleManifest(w.fs, w.modStore, mod.Path)
							},
							Type:  op.OpTypeParseModuleManifest.String(),
							Defer: decodeCalledModulesFunc(w.fs, w.modStore, w.schemaStore, w, mod.Path),
						})
						if err == nil {
							w.jobStore.WaitForJobs(ctx, id)
							collectReferences(modHandle, w.modStore, w.schemaStore)(ctx, nil)
						}

						return nil
					}
					if containsPath(mod.Watchable.PluginLockFiles, path) {
						w.jobStore.EnqueueJob(job.Job{
							Dir: modHandle,
							Func: func(ctx context.Context) error {
								ctx = exec.WithExecutorFactory(ctx, w.tfExecFactory)
								eo, ok := exec.ExecutorOptsFromContext(ctx)
								if ok {
									ctx = exec.WithExecutorOpts(ctx, eo)
								}

								return ObtainSchema(ctx, w.modStore, w.schemaStore, mod.Path)
							},
							Type: op.OpTypeObtainSchema.String(),
						})
						w.jobStore.EnqueueJob(job.Job{
							Dir: modHandle,
							Func: func(ctx context.Context) error {
								ctx = exec.WithExecutorFactory(ctx, w.tfExecFactory)
								eo, ok := exec.ExecutorOptsFromContext(ctx)
								if ok {
									ctx = exec.WithExecutorOpts(ctx, eo)
								}

								return GetTerraformVersion(ctx, w.modStore, mod.Path)
							},
							Type: op.OpTypeGetTerraformVersion.String(),
						})
						return nil
					}
					return nil
				})

				return
			}

			if containsPath(mod.Watchable.ModuleManifests, eventPath) {
				id, err := w.jobStore.EnqueueJob(job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						return ParseModuleManifest(w.fs, w.modStore, mod.Path)
					},
					Type:  op.OpTypeParseModuleManifest.String(),
					Defer: decodeCalledModulesFunc(w.fs, w.modStore, w.schemaStore, w, mod.Path),
				})
				if err == nil {
					w.jobStore.WaitForJobs(ctx, id)
					collectReferences(modHandle, w.modStore, w.schemaStore)(ctx, nil)
				}
				return
			}

			if containsPath(mod.Watchable.PluginLockFiles, eventPath) {
				w.jobStore.EnqueueJob(job.Job{
					Dir: modHandle,
					Func: func(jCtx context.Context) error {
						ctx = exec.WithExecutorFactory(ctx, w.tfExecFactory)
						eo, ok := exec.ExecutorOptsFromContext(ctx)
						if ok {
							ctx = exec.WithExecutorOpts(ctx, eo)
						}

						return ObtainSchema(ctx, w.modStore, w.schemaStore, mod.Path)
					},
					Type: op.OpTypeObtainSchema.String(),
				})
				w.jobStore.EnqueueJob(job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						ctx = exec.WithExecutorFactory(ctx, w.tfExecFactory)
						eo, ok := exec.ExecutorOptsFromContext(ctx)
						if ok {
							ctx = exec.WithExecutorOpts(ctx, eo)
						}

						return GetTerraformVersion(ctx, w.modStore, mod.Path)
					},
					Type: op.OpTypeGetTerraformVersion.String(),
				})
				return
			}
		}
	}

	if event.Op&fsnotify.Remove == fsnotify.Remove {
		for modI, mod := range w.modules {
			// Whole module being removed
			if pathcmp.PathEquals(mod.Path, eventPath) {
				for _, wPath := range mod.Watched {
					w.fw.Remove(wPath)
				}
				w.fw.Remove(mod.Path)
				w.modules = append(w.modules[:modI], w.modules[modI+1:]...)
				return
			}

			for i, wp := range mod.Watched {
				if pathcmp.PathEquals(wp, eventPath) {
					w.fw.Remove(wp)
					mod.Watched = append(mod.Watched[:i], mod.Watched[i+1:]...)
					return
				}
			}
		}
	}
}

func decodeCalledModulesFunc(fs ReadOnlyFS, modStore *state.ModuleStore, schemaReader state.SchemaReader, w Watcher, modPath string) job.DeferFunc {
	return func(ctx context.Context, opErr error) (jobIds job.IDs) {
		if opErr != nil {
			return
		}

		moduleCalls, err := modStore.ModuleCalls(modPath)
		if err != nil {
			return
		}

		jobStore, err := job.JobStoreFromContext(ctx)
		if err != nil {
			return
		}

		for _, mc := range moduleCalls {
			fi, err := os.Stat(mc.Path)
			if err != nil || !fi.IsDir() {
				continue
			}
			modStore.Add(mc.Path)

			mcHandle := document.DirHandleFromPath(mc.Path)

			id, err := jobStore.EnqueueJob(job.Job{
				Dir: mcHandle,
				Func: func(ctx context.Context) error {
					return ParseModuleConfiguration(fs, modStore, mc.Path)
				},
				Type: op.OpTypeParseModuleConfiguration.String(),
				Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
					id, err := jobStore.EnqueueJob(job.Job{
						Dir:  mcHandle,
						Type: op.OpTypeLoadModuleMetadata.String(),
						Func: func(ctx context.Context) error {
							return LoadModuleMetadata(modStore, mc.Path)
						},
						Defer: collectReferences(mcHandle, modStore, schemaReader),
					})
					if err != nil {
						return
					}
					ids = append(ids, id)
					return
				},
			})
			if err != nil {
				return
			}
			jobIds = append(jobIds, id)

			id, err = jobStore.EnqueueJob(job.Job{
				Dir: mcHandle,
				Func: func(ctx context.Context) error {
					return ParseVariables(fs, modStore, mc.Path)
				},
				Type: op.OpTypeParseVariables.String(),
			})
			if err != nil {
				return
			}
			jobIds = append(jobIds, id)

			if w != nil {
				w.AddModule(mc.Path)
			}
		}

		return
	}
}

func collectReferences(dirHandle document.DirHandle, modStore *state.ModuleStore, schemaReader state.SchemaReader) job.DeferFunc {
	return func(ctx context.Context, jobErr error) (ids job.IDs) {
		jobStore, err := job.JobStoreFromContext(ctx)
		if err != nil {
			return
		}

		id, err := jobStore.EnqueueJob(job.Job{
			Dir: dirHandle,
			Func: func(ctx context.Context) error {
				return DecodeReferenceTargets(ctx, modStore, schemaReader, dirHandle.Path())
			},
			Type: op.OpTypeDecodeReferenceTargets.String(),
		})
		if err != nil {
			return
		}
		ids = append(ids, id)

		id, err = jobStore.EnqueueJob(job.Job{
			Dir: dirHandle,
			Func: func(ctx context.Context) error {
				return DecodeReferenceOrigins(ctx, modStore, schemaReader, dirHandle.Path())
			},
			Type: op.OpTypeDecodeReferenceOrigins.String(),
		})
		if err != nil {
			return
		}
		ids = append(ids, id)

		id, err = jobStore.EnqueueJob(job.Job{
			Dir: dirHandle,
			Func: func(ctx context.Context) error {
				return DecodeVarsReferences(ctx, modStore, schemaReader, dirHandle.Path())
			},
			Type: op.OpTypeDecodeVarsReferences.String(),
		})
		if err != nil {
			return
		}
		ids = append(ids, id)

		return
	}
}

func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if pathcmp.PathEquals(p, path) {
			return true
		}
	}
	return false
}

func (w *watcher) Start(ctx context.Context) error {
	if w.watching {
		w.logger.Println("watching already in progress")
		return nil
	}

	ctx, cancelFunc := context.WithCancel(ctx)
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
