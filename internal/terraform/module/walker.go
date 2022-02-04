package module

import (
	"container/heap"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
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
	fs ReadOnlyFS

	modStore      *state.ModuleStore
	schemaStore   *state.ProviderSchemaStore
	jobStore      job.JobStore
	tfExecFactory exec.ExecutorFactory

	watcher Watcher
	logger  *log.Logger
	sync    bool

	queue    *walkerQueue
	queueMu  *sync.Mutex
	pushChan chan struct{}

	walking    bool
	walkingMu  *sync.RWMutex
	cancelFunc context.CancelFunc
	doneCh     <-chan struct{}

	excludeModulePaths   map[string]bool
	ignoreDirectoryNames map[string]bool
}

// queueCap represents channel buffer size
// which when reached causes EnqueuePath to block
// until a path is consumed
const queueCap = 50

func NewWalker(fs ReadOnlyFS, ds DocumentStore, ms *state.ModuleStore, pss *state.ProviderSchemaStore, js job.JobStore, tfExec exec.ExecutorFactory) *Walker {
	return &Walker{
		fs:                   fs,
		modStore:             ms,
		jobStore:             js,
		schemaStore:          pss,
		tfExecFactory:        tfExec,
		logger:               discardLogger,
		walkingMu:            &sync.RWMutex{},
		queue:                newWalkerQueue(ds),
		queueMu:              &sync.Mutex{},
		pushChan:             make(chan struct{}, queueCap),
		doneCh:               make(chan struct{}, 0),
		ignoreDirectoryNames: skipDirNames,
	}
}

func (w *Walker) SetLogger(logger *log.Logger) {
	w.logger = logger
}

func (w *Walker) SetWatcher(watcher Watcher) {
	w.watcher = watcher
}

func (w *Walker) SetExcludeModulePaths(excludeModulePaths []string) {
	w.excludeModulePaths = make(map[string]bool)
	for _, path := range excludeModulePaths {
		w.excludeModulePaths[path] = true
	}
}

func (w *Walker) SetIgnoreDirectoryNames(ignoreDirectoryNames []string) {
	for _, path := range ignoreDirectoryNames {
		w.ignoreDirectoryNames[path] = true
	}
}

func (w *Walker) Stop() {
	if w.cancelFunc != nil {
		w.cancelFunc()
	}

	if w.IsWalking() {
		w.logger.Println("stopping walker")
		w.setWalking(false)
	}
}

func (w *Walker) setWalking(isWalking bool) {
	w.walkingMu.Lock()
	defer w.walkingMu.Unlock()
	w.walking = isWalking
}

func (w *Walker) EnqueuePath(path string) {
	w.queueMu.Lock()
	defer w.queueMu.Unlock()
	heap.Push(w.queue, path)

	w.triggerConsumption()
}

func (w *Walker) triggerConsumption() {
	w.pushChan <- struct{}{}
}

func (w *Walker) RemovePathFromQueue(path string) {
	w.queueMu.Lock()
	defer w.queueMu.Unlock()
	w.queue.RemoveFromQueue(path)
}

func (w *Walker) StartWalking(ctx context.Context) error {
	if w.IsWalking() {
		return fmt.Errorf("walker is already running")
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	w.cancelFunc = cancelFunc
	w.doneCh = ctx.Done()
	w.setWalking(true)

	if w.sync {
		var errs *multierror.Error

		jobIds := make(job.IDs, 0)
		for {
			w.queueMu.Lock()
			if w.queue.Len() == 0 {
				w.queueMu.Unlock()
				break
			}
			nextPath := heap.Pop(w.queue)
			w.queueMu.Unlock()

			path := nextPath.(string)
			w.logger.Printf("synchronously walking through %s", path)
			ids, err := w.walk(ctx, path)
			if err != nil {
				multierror.Append(errs, err)
			}
			jobIds = append(jobIds, ids...)
		}

		w.logger.Printf("waiting for %d jobs before stopping walker", len(jobIds))
		err := w.jobStore.WaitForJobs(ctx, jobIds...)
		if err != nil {
			return err
		}
		w.logger.Println("waiting done, synchronous walking finished")

		w.Stop()
		return errs.ErrorOrNil()
	}

	var nextPathToWalk = make(chan string)

	go func(w *Walker) {
		for {
			w.queueMu.Lock()
			if w.queue.Len() == 0 {
				w.queueMu.Unlock()
				select {
				case <-w.pushChan:
					// block to avoid infinite loop
					continue
				case <-w.doneCh:
					return
				}
			}
			nextPath := heap.Pop(w.queue)
			w.queueMu.Unlock()
			path := nextPath.(string)
			nextPathToWalk <- path
		}
	}(w)

	go func(w *Walker, pathsChan chan string) {
		for {
			select {
			case <-w.doneCh:
				return
			case path := <-pathsChan:
				w.logger.Printf("asynchronously walking through %s", path)
				_, err := w.walk(ctx, path)
				if err != nil {
					w.logger.Printf("async walking through %s failed: %s", path, err)
					return
				}
				w.logger.Printf("async walking through %s finished", path)
			}
		}
	}(w, nextPathToWalk)

	return nil
}

func (w *Walker) IsWalking() bool {
	w.walkingMu.RLock()
	defer w.walkingMu.RUnlock()

	return w.walking
}

func (w *Walker) isSkippableDir(dirName string) bool {
	_, ok := w.ignoreDirectoryNames[dirName]
	return ok
}

func (w *Walker) walk(ctx context.Context, rootPath string) (jobIds job.IDs, err error) {
	// We ignore the passed FS and instead read straight from OS FS
	// because that would require reimplementing filepath.Walk and
	// the data directory should never be on the virtual filesystem anyway
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
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

			_, err := w.modStore.ModuleByPath(dir)
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

			id, err := w.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return ParseModuleConfiguration(w.fs, w.modStore, dir)
				},
				Type: op.OpTypeParseModuleConfiguration.String(),
			})
			if err != nil {
				return err
			}
			jobIds = append(jobIds, id)

			id, err = w.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return ParseVariables(w.fs, w.modStore, dir)
				},
				Type: op.OpTypeParseVariables.String(),
			})
			if err != nil {
				return err
			}
			jobIds = append(jobIds, id)

			id, err = w.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					ctx = exec.WithExecutorFactory(ctx, w.tfExecFactory)
					return GetTerraformVersion(ctx, w.modStore, dir)
				},
				Type: op.OpTypeGetTerraformVersion.String(),
			})
			if err != nil {
				return err
			}
			jobIds = append(jobIds, id)

			dataDir := datadir.WalkDataDirOfModule(w.fs, dir)
			w.logger.Printf("parsed datadir: %#v", dataDir)
			if dataDir.ModuleManifestPath != "" {
				// References are collected *after* manifest parsing
				// so that we reflect any references to submodules.
				id, err = w.jobStore.EnqueueJob(job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						return ParseModuleManifest(w.fs, w.modStore, dir)
					},
					Type:  op.OpTypeParseModuleManifest.String(),
					Defer: decodeCalledModulesFunc(w.fs, w.modStore, w.schemaStore, w.watcher, dir),
				})
				if err != nil {
					return err
				}

				// Here we wait for all module calls to be processed to
				// reflect any metadata required to collect reference origins.
				// This assumes scheduler is running to consume the jobs
				// by the time we reach this point.
				w.jobStore.WaitForJobs(ctx, id)
			}

			id, err = w.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return DecodeReferenceTargets(ctx, w.modStore, w.schemaStore, dir)
				},
				Type: op.OpTypeDecodeReferenceTargets.String(),
			})
			if err != nil {
				return err
			}
			jobIds = append(jobIds, id)

			id, err = w.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return DecodeReferenceOrigins(ctx, w.modStore, w.schemaStore, dir)
				},
				Type: op.OpTypeDecodeReferenceOrigins.String(),
			})
			if err != nil {
				return err
			}
			jobIds = append(jobIds, id)

			id, err = w.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return DecodeVarsReferences(ctx, w.modStore, w.schemaStore, dir)
				},
				Type: op.OpTypeDecodeVarsReferences.String(),
			})
			if err != nil {
				return err
			}
			jobIds = append(jobIds, id)

			if dataDir.PluginLockFilePath != "" {
				id, err = w.jobStore.EnqueueJob(job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						ctx = exec.WithExecutorFactory(ctx, w.tfExecFactory)
						return ObtainSchema(ctx, w.modStore, w.schemaStore, dir)
					},
					Type: op.OpTypeObtainSchema.String(),
				})
				if err != nil {
					return err
				}
				jobIds = append(jobIds, id)
			}

			if w.watcher != nil {
				w.watcher.AddModule(dir)
			}

			return nil
		}

		if !info.IsDir() {
			// All files are skipped, we only care about dirs
			return nil
		}

		return nil
	})
	w.logger.Printf("walking of %s finished", rootPath)
	return
}
