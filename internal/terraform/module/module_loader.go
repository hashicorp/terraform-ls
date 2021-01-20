package module

import (
	"context"
	"log"
	"runtime"
	"sync/atomic"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

type moduleLoader struct {
	queue              moduleOpsQueue
	nonPrioParallelism int64
	prioParallelism    int64
	logger             *log.Logger
	tfExecOpts         *exec.ExecutorOpts
	opsToDispatch      chan ModuleOperation

	loadingCount     *int64
	prioLoadingCount *int64
}

func newModuleLoader() *moduleLoader {
	nonPrioParallelism := 2 * runtime.NumCPU()
	prioParallelism := 1 * runtime.NumCPU()

	plc, lc := int64(0), int64(0)
	ml := &moduleLoader{
		queue:              newModuleOpsQueue(),
		logger:             defaultLogger,
		nonPrioParallelism: int64(nonPrioParallelism),
		prioParallelism:    int64(prioParallelism),
		opsToDispatch:      make(chan ModuleOperation, 1),
		loadingCount:       &lc,
		prioLoadingCount:   &plc,
	}

	return ml
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
				mod := ml.queue.PopOp()
				go func(ml *moduleLoader) {
					defer atomic.AddInt64(ml.prioLoadingCount, -1)
					ml.executeModuleOp(ctx, mod)
				}(ml)
			} else if ml.nonPrioCapacity() > 0 {
				atomic.AddInt64(ml.loadingCount, 1)
				mod := ml.queue.PopOp()
				go func(ml *moduleLoader) {
					defer atomic.AddInt64(ml.loadingCount, -1)
					ml.executeModuleOp(ctx, mod)
				}(ml)
			}
		}
	}
}

func (ml *moduleLoader) tryDispatchingModuleOp() {
	totalCapacity := ml.nonPrioCapacity() + ml.prioCapacity()
	opsInQueue := ml.queue.Len()

	// Keep scheduling work from queue if we have capacity
	if opsInQueue > 0 && totalCapacity > 0 {
		item := ml.queue.Peek()
		nextModOp := item.(ModuleOperation)
		ml.opsToDispatch <- nextModOp
	}
}

func (ml *moduleLoader) prioCapacity() int64 {
	return ml.prioParallelism - atomic.LoadInt64(ml.prioLoadingCount)
}

func (ml *moduleLoader) nonPrioCapacity() int64 {
	return ml.prioParallelism - atomic.LoadInt64(ml.loadingCount)
}

func (ml *moduleLoader) executeModuleOp(ctx context.Context, modOp ModuleOperation) {
	ml.logger.Printf("executing %q for %s", modOp.Type, modOp.Module.Path())
	// TODO: Report progress in % for each op based on queue length
	defer ml.logger.Printf("finished %q for %s", modOp.Type, modOp.Module.Path())
	defer modOp.markAsDone()
	defer ml.tryDispatchingModuleOp()

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

	ml.tryDispatchingModuleOp()
}
