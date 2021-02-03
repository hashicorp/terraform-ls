package module

import (
	"context"
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type moduleLoader struct {
	queue              moduleOpsQueue
	nonPrioParallelism int64
	prioParallelism    int64
	logger             *log.Logger
	tfExecOpts         *exec.ExecutorOpts
	opsToDispatch      chan ModuleOperation
	reportProgress     ProgressReporterFunc

	loadingCount     *int64
	prioLoadingCount *int64
}

func newModuleLoader() *moduleLoader {
	p := loaderParallelism(runtime.NumCPU())
	plc, lc := int64(0), int64(0)
	ml := &moduleLoader{
		queue:              newModuleOpsQueue(),
		logger:             defaultLogger,
		nonPrioParallelism: p.NonPriority,
		prioParallelism:    p.Priority,
		opsToDispatch:      make(chan ModuleOperation, 1),
		loadingCount:       &lc,
		prioLoadingCount:   &plc,
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

			if nextOp.Module.HasOpenFiles() && ml.prioCapacity() > 0 {
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

func (ml *moduleLoader) SetProgressReporter(f ProgressReporterFunc) {
	ml.reportProgress = f
}

func (ml *moduleLoader) executeModuleOp(ctx context.Context, modOp ModuleOperation) {
	ml.logger.Printf("executing %q for %s", modOp.Type, modOp.Module.Path())

	if ml.reportProgress != nil {
		go ml.reportProgress(ctx, modOp, OpStateLoading)
	}

	defer func(ml *moduleLoader, modOp ModuleOperation) {
		ml.logger.Printf("finished %q for %s", modOp.Type, modOp.Module.Path())
		modOp.markAsDone()
		if ml.reportProgress != nil {
			go ml.reportProgress(ctx, modOp, OpStateLoaded)
		}
	}(ml, modOp)

	switch modOp.Type {
	case OpTypeGetTerraformVersion:
		GetTerraformVersion(ctx, modOp.Module)
		return
	case OpTypeObtainSchema:
		ObtainSchema(ctx, modOp.Module)
		return
	case OpTypeParseConfiguration:
		ParseConfiguration(modOp.Module)
		return
	case OpTypeParseModuleManifest:
		ParseModuleManifest(modOp.Module)
		return
	}

	ml.logger.Printf("%s: unknown operation (%#v) for module operation",
		modOp.Module.Path(), modOp.Type)
}

func (ml *moduleLoader) EnqueueModuleOp(modOp ModuleOperation) {
	m := modOp.Module
	mod := m.(*module)

	ml.logger.Printf("ML: enqueing %q module operation: %s", modOp.Type, mod.Path())

	switch modOp.Type {
	case OpTypeGetTerraformVersion:
		if mod.TerraformVersionState() == OpStateQueued {
			// avoid enqueuing duplicate operation
			return
		}
		mod.SetTerraformVersionState(OpStateQueued)
	case OpTypeObtainSchema:
		if mod.ProviderSchemaState() == OpStateQueued {
			// avoid enqueuing duplicate operation
			return
		}
		mod.SetProviderSchemaObtainingState(OpStateQueued)
	case OpTypeParseConfiguration:
		if mod.ConfigParsingState() == OpStateQueued {
			// avoid enqueuing duplicate operation
			return
		}
		mod.SetConfigParsingState(OpStateQueued)
	case OpTypeParseModuleManifest:
		if mod.ModuleManifestState() == OpStateQueued {
			// avoid enqueuing duplicate operation
			return
		}
		mod.SetModuleManifestParsingState(OpStateQueued)
	}

	ml.queue.PushOp(modOp)

	if ml.reportProgress != nil {
		go ml.reportProgress(context.Background(), modOp, OpStateQueued)
	}

	ml.tryDispatchingModuleOp()
}
