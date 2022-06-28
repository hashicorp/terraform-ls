package module

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

	pathStore     PathStore
	modStore      *state.ModuleStore
	schemaStore   *state.ProviderSchemaStore
	jobStore      job.JobStore
	tfExecFactory exec.ExecutorFactory

	logger *log.Logger

	Collector *WalkerCollector

	cancelFunc context.CancelFunc

	excludeModulePaths   map[string]bool
	ignoreDirectoryNames map[string]bool
}

type PathStore interface {
	AwaitNextDir(ctx context.Context) (document.DirHandle, error)
	RemoveDir(dir document.DirHandle) error
}

func NewWalker(fs ReadOnlyFS, ps PathStore, ms *state.ModuleStore, pss *state.ProviderSchemaStore, js job.JobStore, tfExec exec.ExecutorFactory) *Walker {
	return &Walker{
		pathStore:            ps,
		fs:                   fs,
		modStore:             ms,
		jobStore:             js,
		schemaStore:          pss,
		tfExecFactory:        tfExec,
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

func (w *Walker) collectJobId(jobId job.ID) {
	if w.Collector != nil {
		w.Collector.CollectJobId(jobId)
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

			// TODO: decouple into its own function and pass as WalkFunc to NewWalker
			// this could reduce the amount of args in NewWalker

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
			w.collectJobId(id)

			id, err = w.jobStore.EnqueueJob(job.Job{
				Dir: modHandle,
				Func: func(ctx context.Context) error {
					return ParseVariables(w.fs, w.modStore, dir)
				},
				Type: op.OpTypeParseVariables.String(),
				Defer: func(ctx context.Context, jobErr error) (ids job.IDs) {
					id, err := w.jobStore.EnqueueJob(job.Job{
						Dir: modHandle,
						Func: func(ctx context.Context) error {
							return DecodeVarsReferences(ctx, w.modStore, w.schemaStore, dir)
						},
						Type: op.OpTypeDecodeVarsReferences.String(),
					})
					if err != nil {
						return
					}
					ids = append(ids, id)
					return
				},
			})
			if err != nil {
				return err
			}
			w.collectJobId(id)

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
			w.collectJobId(id)

			dataDir := datadir.WalkDataDirOfModule(w.fs, dir)
			w.logger.Printf("parsed datadir: %#v", dataDir)

			if dataDir.PluginLockFilePath != "" {
				id, err := w.jobStore.EnqueueJob(job.Job{
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
				w.collectJobId(id)
			}

			if dataDir.ModuleManifestPath != "" {
				// References are collected *after* manifest parsing
				// so that we reflect any references to submodules.
				id, err := w.jobStore.EnqueueJob(job.Job{
					Dir: modHandle,
					Func: func(ctx context.Context) error {
						return ParseModuleManifest(w.fs, w.modStore, dir)
					},
					Type:  op.OpTypeParseModuleManifest.String(),
					Defer: decodeInstalledModuleCalls(w.fs, w.modStore, w.schemaStore, dir),
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
			w.collectJobId(id)

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
			w.collectJobId(id)

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
