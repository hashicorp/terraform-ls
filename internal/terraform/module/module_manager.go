package module

import (
	"context"
	"log"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/state"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfmodule "github.com/hashicorp/terraform-schema/module"
)

type moduleManager struct {
	fs          ReadOnlyFS
	moduleStore *state.ModuleStore
	schemaStore *state.ProviderSchemaStore

	loader      *moduleLoader
	syncLoading bool
	cancelFunc  context.CancelFunc
	logger      *log.Logger
}

func NewModuleManager(ctx context.Context, fs ReadOnlyFS, ds DocumentStore, ms *state.ModuleStore, pss *state.ProviderSchemaStore) ModuleManager {
	mm := newModuleManager(fs, ds, ms, pss)

	ctx, cancelFunc := context.WithCancel(ctx)
	mm.cancelFunc = cancelFunc
	mm.loader.Start(ctx)

	return mm
}

func NewSyncModuleManager(ctx context.Context, fs ReadOnlyFS, ds DocumentStore, ms *state.ModuleStore, pss *state.ProviderSchemaStore) ModuleManager {
	mm := newModuleManager(fs, ds, ms, pss)

	ctx, cancelFunc := context.WithCancel(ctx)
	mm.cancelFunc = cancelFunc
	mm.syncLoading = true

	mm.loader.Start(ctx)

	return mm
}

func newModuleManager(fs ReadOnlyFS, ds DocumentStore, ms *state.ModuleStore, pss *state.ProviderSchemaStore) *moduleManager {
	mm := &moduleManager{
		fs:          fs,
		moduleStore: ms,
		schemaStore: pss,
		logger:      defaultLogger,
		loader:      newModuleLoader(fs, ds, ms, pss),
	}
	return mm
}

func (mm *moduleManager) SetLogger(logger *log.Logger) {
	mm.logger = logger
	mm.loader.SetLogger(logger)
}

func (mm *moduleManager) AddModule(modPath string) (Module, error) {
	modPath = filepath.Clean(modPath)

	mm.logger.Printf("MM: adding new module: %s", modPath)
	// TODO: Follow symlinks (requires proper test data)

	err := mm.moduleStore.Add(modPath)
	if err != nil {
		if _, ok := err.(*state.AlreadyExistsError); !ok {
			return nil, err
		}
	}

	// TODO: Avoid returning new module, just the error from adding
	mod, err := mm.moduleStore.ModuleByPath(modPath)
	return mod, err
}

func (mm *moduleManager) RemoveModule(modPath string) error {
	mm.loader.DequeueModule(modPath)
	return mm.moduleStore.Remove(modPath)
}

func (mm *moduleManager) EnqueueModuleOp(modPath string, opType op.OpType, deferFunc DeferFunc) error {
	modOp := NewModuleOperation(modPath, opType)
	modOp.Defer = deferFunc
	mm.loader.EnqueueModuleOp(modOp)
	if mm.syncLoading {
		<-modOp.done()
	}
	return nil
}

func (mm *moduleManager) CallersOfModule(modPath string) ([]Module, error) {
	modules := make([]Module, 0)
	callers, err := mm.moduleStore.CallersOfModule(modPath)
	if err != nil {
		return modules, err
	}

	for _, mod := range callers {
		modules = append(modules, mod)
	}

	return modules, nil
}

func (mm *moduleManager) ModuleCalls(modPath string) ([]tfmodule.ModuleCall, error) {
	return mm.moduleStore.ModuleCalls(modPath)
}

// SchemaSourcesForModule is DEPRECATED and should NOT be used anymore
// it is just maintained for backwards compatibility in the "rootmodules"
// custom LSP command which itself will be DEPRECATED as external parties
// should not need to know where does a matched schema come from in practice
func (mm *moduleManager) SchemaSourcesForModule(modPath string) ([]SchemaSource, error) {
	ok, err := mm.moduleHasAnyLocallySourcedSchema(modPath)
	if err != nil {
		return nil, err
	}
	if ok {
		return []SchemaSource{
			{Path: modPath},
		}, nil
	}

	callers, err := mm.moduleStore.CallersOfModule(modPath)
	if err != nil {
		return nil, err
	}

	sources := make([]SchemaSource, 0)
	for _, modCaller := range callers {
		ok, err := mm.moduleHasAnyLocallySourcedSchema(modCaller.Path)
		if err != nil {
			return nil, err
		}
		if ok {
			sources = append(sources, SchemaSource{
				Path: modCaller.Path,
			})
		}

	}

	return sources, nil
}

func (mm *moduleManager) moduleHasAnyLocallySourcedSchema(modPath string) (bool, error) {
	si, err := mm.schemaStore.ListSchemas()
	if err != nil {
		return false, err
	}

	for ps := si.Next(); ps != nil; ps = si.Next() {
		if lss, ok := ps.Source.(state.LocalSchemaSource); ok {
			if lss.ModulePath == modPath {
				return true, nil
			}
		}
	}

	return false, nil
}

func (mm *moduleManager) ListModules() ([]Module, error) {
	modules := make([]Module, 0)

	mods, err := mm.moduleStore.List()
	if err != nil {
		return modules, err
	}

	for _, mod := range mods {
		modules = append(modules, mod)
	}

	return modules, nil
}
func (mm *moduleManager) ModuleByPath(path string) (Module, error) {
	path = filepath.Clean(path)

	mod, err := mm.moduleStore.ModuleByPath(path)
	if err != nil {
		if _, ok := err.(*state.ModuleNotFoundError); ok {
			return nil, &ModuleNotFoundErr{path}
		}
		return nil, err
	}

	return mod, nil
}

func (mm *moduleManager) CancelLoading() {
	mm.cancelFunc()
}
