package module

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/schema"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/pathcmp"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

type moduleManager struct {
	modules []*module
	fs      filesystem.Filesystem

	loader      *moduleLoader
	syncLoading bool
	cancelFunc  context.CancelFunc
	logger      *log.Logger
}

func NewModuleManager(ctx context.Context, fs filesystem.Filesystem) ModuleManager {
	mm := newModuleManager(fs)

	ctx, cancelFunc := context.WithCancel(ctx)
	mm.cancelFunc = cancelFunc
	mm.loader.Start(ctx)

	return mm
}

func NewSyncModuleManager(ctx context.Context, fs filesystem.Filesystem) ModuleManager {
	mm := newModuleManager(fs)

	ctx, cancelFunc := context.WithCancel(ctx)
	mm.cancelFunc = cancelFunc
	mm.syncLoading = true

	mm.loader.Start(ctx)

	return mm
}

func newModuleManager(fs filesystem.Filesystem) *moduleManager {
	mm := &moduleManager{
		modules: make([]*module, 0),
		fs:      fs,
		logger:  defaultLogger,
		loader:  newModuleLoader(),
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

	if _, ok := mm.moduleByPath(modPath); ok {
		return nil, fmt.Errorf("module %s was already added", modPath)
	}

	mod := newModule(mm.fs, modPath)
	mod.SetLogger(mm.logger)

	mm.modules = append(mm.modules, mod)

	return mod, nil
}

func (mm *moduleManager) EnqueueModuleOpWait(modPath string, opType OpType) error {
	mod, err := mm.ModuleByPath(modPath)
	if err != nil {
		return err
	}
	modOp := NewModuleOperation(mod, opType)
	mm.loader.EnqueueModuleOp(modOp)

	<-modOp.Done()

	return nil
}

func (mm *moduleManager) EnqueueModuleOp(modPath string, opType OpType) error {
	mod, err := mm.ModuleByPath(modPath)
	if err != nil {
		return err
	}

	modOp := NewModuleOperation(mod, opType)
	mm.loader.EnqueueModuleOp(modOp)
	if mm.syncLoading {
		<-modOp.Done()
	}
	return nil
}

func (mm *moduleManager) SchemaForModule(modPath string) (*schema.BodySchema, error) {
	sources, err := mm.SchemaSourcesForModule(modPath)
	if err != nil {
		return nil, err
	}

	var (
		tfVersion        *version.Version
		coreSchema       *schema.BodySchema
		providerSchema   *tfjson.ProviderSchemas
		providerVersions map[string]*version.Version
	)

	if len(sources) > 0 {
		ps, err := sources[0].ProviderSchema()
		if err == nil {
			providerSchema = ps
		}
		providerVersions = sources[0].ProviderVersions()
	}

	mod, err := mm.ModuleByPath(modPath)
	if err != nil {
		return nil, err
	}

	if v, err := mod.TerraformVersion(); err == nil {
		tfVersion = v
	}

	if len(sources) == 0 {
		mm.logger.Printf("falling back to preloaded schema for %s...", modPath)
		ps, vOut, err := schemas.PreloadedProviderSchemas()
		if err != nil {
			return nil, err
		}
		if ps != nil {
			providerSchema = ps
			providerVersions = vOut.Providers

			mm.logger.Printf("preloaded provider schema (%d providers) set for %s",
				len(ps.Schemas), modPath)

			if tfVersion == nil {
				tfVersion = vOut.Core
			}
		}
	}

	if tfVersion != nil {
		coreSchema, err = tfschema.CoreModuleSchemaForVersion(tfVersion)
		if err != nil {
			return nil, err
		}
	} else {
		coreSchema = tfschema.UniversalCoreModuleSchema()
	}

	merger := tfschema.NewSchemaMerger(coreSchema)
	if tfVersion != nil {
		merger.SetCoreVersion(tfVersion)
	}
	if len(providerVersions) > 0 {
		err = merger.SetProviderVersions(providerVersions)
		if err != nil {
			return nil, err
		}
	}

	pf, _ := mod.ParsedFiles()
	if len(pf) > 0 {
		merger.SetParsedFiles(pf)
	}

	return merger.MergeWithJsonProviderSchemas(providerSchema)
}

func (mm *moduleManager) SchemaSourcesForModule(modPath string) ([]SchemaSource, error) {
	mod, err := mm.ModuleByPath(modPath)
	if err != nil {
		return []SchemaSource{}, err
	}

	if ps, err := mod.ProviderSchema(); err == nil && ps != nil {
		return []SchemaSource{mod}, nil
	}

	sources := make([]SchemaSource, 0)
	for _, mod := range mm.modules {
		if mod.CallsModule(modPath) {
			if ps, err := mod.ProviderSchema(); err == nil && ps != nil {
				sources = append(sources, mod)
			}
		}
	}

	// We could expose preloaded schemas here already
	// but other logic elsewhere isn't able to take advantage
	// of multiple sources and mix-and-match yet.
	// TODO https://github.com/hashicorp/terraform-ls/issues/354

	return sources, nil
}

func (mm *moduleManager) moduleByPath(dir string) (*module, bool) {
	for _, mod := range mm.modules {
		if pathcmp.PathEquals(mod.Path(), dir) {
			return mod, true
		}
	}
	return nil, false
}

func (mm *moduleManager) ListModules() []Module {
	modules := make([]Module, 0)
	for _, mod := range mm.modules {
		modules = append(modules, mod)
	}
	return modules
}
func (mm *moduleManager) ModuleByPath(path string) (Module, error) {
	path = filepath.Clean(path)

	if mod, ok := mm.moduleByPath(path); ok {
		return mod, nil
	}

	return nil, &ModuleNotFoundErr{path}
}

func (mm *moduleManager) CancelLoading() {
	mm.cancelFunc()
}
