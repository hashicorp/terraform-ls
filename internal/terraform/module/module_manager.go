package module

import (
	"context"
	"log"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/state"
	op "github.com/hashicorp/terraform-ls/internal/terraform/module/operation"
	tfmodule "github.com/hashicorp/terraform-schema/module"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

type moduleManager struct {
	fs          filesystem.Filesystem
	moduleStore *state.ModuleStore
	schemaStore *state.ProviderSchemaStore

	loader      *moduleLoader
	syncLoading bool
	cancelFunc  context.CancelFunc
	logger      *log.Logger
}

func NewModuleManager(ctx context.Context, fs filesystem.Filesystem, ms *state.ModuleStore, pss *state.ProviderSchemaStore) ModuleManager {
	mm := newModuleManager(fs, ms, pss)

	ctx, cancelFunc := context.WithCancel(ctx)
	mm.cancelFunc = cancelFunc
	mm.loader.Start(ctx)

	return mm
}

func NewSyncModuleManager(ctx context.Context, fs filesystem.Filesystem, ms *state.ModuleStore, pss *state.ProviderSchemaStore) ModuleManager {
	mm := newModuleManager(fs, ms, pss)

	ctx, cancelFunc := context.WithCancel(ctx)
	mm.cancelFunc = cancelFunc
	mm.syncLoading = true

	mm.loader.Start(ctx)

	return mm
}

func newModuleManager(fs filesystem.Filesystem, ms *state.ModuleStore, pss *state.ProviderSchemaStore) *moduleManager {
	mm := &moduleManager{
		fs:          fs,
		moduleStore: ms,
		schemaStore: pss,
		logger:      defaultLogger,
		loader:      newModuleLoader(fs, ms, pss),
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
		return nil, err
	}

	// TODO: Avoid returning new module, just the error from adding
	mod, err := mm.moduleStore.ModuleByPath(modPath)
	return mod, err
}

func (mm *moduleManager) EnqueueModuleOpWait(modPath string, opType op.OpType) error {
	modOp := NewModuleOperation(modPath, opType)
	mm.loader.EnqueueModuleOp(modOp)

	<-modOp.Done()

	return nil
}

func (mm *moduleManager) EnqueueModuleOp(modPath string, opType op.OpType) error {
	modOp := NewModuleOperation(modPath, opType)
	mm.loader.EnqueueModuleOp(modOp)
	if mm.syncLoading {
		<-modOp.Done()
	}
	return nil
}

func (mm *moduleManager) SchemaForModule(modPath string) (*schema.BodySchema, error) {
	mod, err := mm.ModuleByPath(modPath)
	if err != nil {
		return nil, err
	}

	return schemaForModule(mod, mm.schemaStore)
}

func schemaForModule(mod *state.Module, schemaReader state.SchemaReader) (*schema.BodySchema, error) {
	var coreSchema *schema.BodySchema
	coreRequirements := make(version.Constraints, 0)
	if mod.TerraformVersion != nil {
		var err error
		coreSchema, err = tfschema.CoreModuleSchemaForVersion(mod.TerraformVersion)
		if err != nil {
			return nil, err
		}
		coreRequirements, err = version.NewConstraint(mod.TerraformVersion.String())
		if err != nil {
			return nil, err
		}
	} else {
		coreSchema = tfschema.UniversalCoreModuleSchema()
	}

	sm := tfschema.NewSchemaMerger(coreSchema)
	sm.SetSchemaReader(schemaReader)

	meta := &tfmodule.Meta{
		Path:                 mod.Path,
		CoreRequirements:     coreRequirements,
		ProviderRequirements: mod.Meta.ProviderRequirements,
		ProviderReferences:   mod.Meta.ProviderReferences,
	}

	return sm.SchemaForModule(meta)
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
