package module

import (
	"context"
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
)

type moduleLoader struct {
	queue              moduleOpsQueue
	nonPrioParallelism int64
	prioParallelism    int64
	logger             *log.Logger
	tfExecOpts         *exec.ExecutorOpts
	opsToDispatch      chan ModuleOperation

	fs          filesystem.Filesystem
	modStore    *state.ModuleStore
	schemaStore *state.ProviderSchemaStore

	loadingCount     *int64
	prioLoadingCount *int64
}

func newModuleLoader(fs filesystem.Filesystem, modStore *state.ModuleStore, schemaStore *state.ProviderSchemaStore) *moduleLoader {
	p := loaderParallelism(runtime.NumCPU())
	plc, lc := int64(0), int64(0)
	ml := &moduleLoader{
		queue:              newModuleOpsQueue(fs),
		logger:             defaultLogger,
		nonPrioParallelism: p.NonPriority,
		prioParallelism:    p.Priority,
		opsToDispatch:      make(chan ModuleOperation, 1),
		loadingCount:       &lc,
		prioLoadingCount:   &plc,
		fs:                 fs,
		modStore:           modStore,
		schemaStore:        schemaStore,
	}

	return ml
}

type parallelism struct {
	NonPriority, Priority int64
}

func loaderParallelism(cpu int) parallelism {
	// Cap utilization for powerful machines
	if cpu >= 4 {
		return parallelism{
			NonPriority: int64(3),
			Priority:    int64(1),
		}
	}
	if cpu == 3 {
		return parallelism{
			NonPriority: int64(2),
			Priority:    int64(1),
		}
	}

	return parallelism{
		NonPriority: 1,
		Priority:    1,
	}
}

func (ml *moduleLoader) SetLogger(logger *log.Logger) {
	ml.logger = logger
}

func (ml *moduleLoader) Start(ctx context.Context) {
	go ml.run(ctx)
}

func (ml *moduleLoader) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			ml.logger.Println("Cancelling module loader...")
			return
		case nextOp, ok := <-ml.opsToDispatch:
			if !ok {
				ml.logger.Println("Failed to get next operation")
				return
			}

			hasOpenFiles, _ := ml.fs.HasOpenFiles(nextOp.ModulePath)

			if hasOpenFiles && ml.prioCapacity() > 0 {
				atomic.AddInt64(ml.prioLoadingCount, 1)
				go func(ml *moduleLoader) {
					ml.executeModuleOp(ctx, nextOp)
					atomic.AddInt64(ml.prioLoadingCount, -1)
					ml.tryDispatchingModuleOp()
				}(ml)
			} else if ml.nonPrioCapacity() > 0 {
				atomic.AddInt64(ml.loadingCount, 1)
				go func(ml *moduleLoader) {
					ml.executeModuleOp(ctx, nextOp)
					atomic.AddInt64(ml.loadingCount, -1)
					ml.tryDispatchingModuleOp()
				}(ml)
			} else {
				// Account for an unlikely situation where next operation
				// was dispatched despite no capacity being available.
				// This may happen when op was received from the channel
				// and dispatcher checked capacity before loading counters
				// were decremented.
				ml.logger.Println("no available capacity, retrying dispatch")
				time.Sleep(100 * time.Millisecond)
				ml.queue.PushOp(nextOp)
				go ml.tryDispatchingModuleOp()
			}
		}
	}
}

func (ml *moduleLoader) tryDispatchingModuleOp() {
	totalCapacity := ml.nonPrioCapacity() + ml.prioCapacity()

	// Keep scheduling work from queue if we have capacity
	if totalCapacity > 0 {
		nextModOp, ok := ml.queue.PopOp()
		if ok {
			ml.opsToDispatch <- nextModOp
		}
	}
}

func (ml *moduleLoader) prioCapacity() int64 {
	return ml.prioParallelism - atomic.LoadInt64(ml.prioLoadingCount)
}

func (ml *moduleLoader) nonPrioCapacity() int64 {
	return ml.nonPrioParallelism - atomic.LoadInt64(ml.loadingCount)
}

func (ml *moduleLoader) executeModuleOp(ctx context.Context, modOp ModuleOperation) {
	ml.logger.Printf("executing %q for %s", modOp.Type, modOp.ModulePath)
	// TODO: Report progress in % for each op based on queue length
	defer ml.logger.Printf("finished %q for %s", modOp.Type, modOp.ModulePath)
	defer modOp.markAsDone()

	switch modOp.Type {
	case op.OpTypeGetTerraformVersion:
		err := GetTerraformVersion(ctx, ml.modStore, modOp.ModulePath)
		if err != nil {
			ml.logger.Printf("failed to get terraform version: %s", err)
		}
		return
	case op.OpTypeObtainSchema:
		err := ObtainSchema(ctx, ml.modStore, ml.schemaStore, modOp.ModulePath)
		if err != nil {
			ml.logger.Printf("failed to obtain schema: %s", err)
		}
		return
	case op.OpTypeParseConfiguration:
		err := ParseConfiguration(ml.fs, ml.modStore, modOp.ModulePath)
		if err != nil {
			ml.logger.Printf("failed to parse configuration: %s", err)
		}
		return
	case op.OpTypeParseModuleManifest:
		err := ParseModuleManifest(ml.fs, ml.modStore, modOp.ModulePath)
		if err != nil {
			ml.logger.Printf("failed to parse module manifest: %s", err)
		}
		return
	case op.OpTypeLoadModuleMetadata:
		err := LoadModuleMetadata(ml.modStore, modOp.ModulePath)
		if err != nil {
			ml.logger.Printf("failed to load module metadata: %s", err)
		}
		return
	}

	ml.logger.Printf("%s: unknown operation (%#v) for module operation",
		modOp.ModulePath, modOp.Type)
}

func (ml *moduleLoader) EnqueueModuleOp(modOp ModuleOperation) error {
	mod, err := ml.modStore.ModuleByPath(modOp.ModulePath)
	if err != nil {
		return err
	}

	ml.logger.Printf("ML: enqueing %q module operation: %s", modOp.Type, modOp.ModulePath)

	switch modOp.Type {
	case op.OpTypeGetTerraformVersion:
		if mod.TerraformVersionState == op.OpStateQueued {
			// avoid enqueuing duplicate operation
			return nil
		}
		ml.modStore.SetTerraformVersionState(modOp.ModulePath, op.OpStateQueued)
	case op.OpTypeObtainSchema:
		if mod.ProviderSchemaState == op.OpStateQueued {
			// avoid enqueuing duplicate operation
			return nil
		}
		ml.modStore.SetProviderSchemaState(modOp.ModulePath, op.OpStateQueued)
	case op.OpTypeParseConfiguration:
		if mod.ParsingState == op.OpStateQueued {
			// avoid enqueuing duplicate operation
			return nil
		}
		ml.modStore.SetParsingState(modOp.ModulePath, op.OpStateQueued)
	case op.OpTypeParseModuleManifest:
		if mod.ModManifestState == op.OpStateQueued {
			// avoid enqueuing duplicate operation
			return nil
		}
		ml.modStore.SetModManifestState(modOp.ModulePath, op.OpStateQueued)
	case op.OpTypeLoadModuleMetadata:
		if mod.MetaState == op.OpStateQueued {
			// avoid enqueuing duplicate operation
			return nil
		}
		ml.modStore.SetMetaState(modOp.ModulePath, op.OpStateQueued)
	}

	ml.queue.PushOp(modOp)
	ml.tryDispatchingModuleOp()

	return nil
}
