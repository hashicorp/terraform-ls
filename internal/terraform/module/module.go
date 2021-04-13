package module

import (
	"log"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/pathcmp"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
)

type module struct {
	path   string
	fs     filesystem.Filesystem
	logger *log.Logger

	// module manifest
	modManifest        *datadir.ModuleManifest
	modManifestErr     error
	modManifestMu      *sync.RWMutex
	modManigestState   OpState
	modManifestStateMu *sync.RWMutex

	// provider schema
	providerSchema        *tfjson.ProviderSchemas
	providerSchemaErr     error
	providerSchemaMu      *sync.RWMutex
	providerSchemaState   OpState
	providerSchemaStateMu *sync.RWMutex

	// terraform exec path
	tfExecPath   string
	tfExecPathMu *sync.RWMutex

	// terraform version
	tfVersion        *version.Version
	tfVersionErr     error
	tfVersionMu      *sync.RWMutex
	tfVersionState   OpState
	tfVersionStateMu *sync.RWMutex

	// provider versions
	providerVersions   map[string]*version.Version
	providerVersionsMu *sync.RWMutex

	// config (HCL) parser
	parsedFiles   map[string]*hcl.File
	parsingErr    error
	parserMu      *sync.RWMutex
	parserState   OpState
	parserStateMu *sync.RWMutex

	// module diagnostics
	diags   map[string]hcl.Diagnostics
	diagsMu *sync.RWMutex
}

func newModule(fs filesystem.Filesystem, dir string) *module {
	return &module{
		path:   dir,
		fs:     fs,
		logger: defaultLogger,

		modManifestMu:         &sync.RWMutex{},
		modManifestStateMu:    &sync.RWMutex{},
		providerSchemaMu:      &sync.RWMutex{},
		providerSchemaStateMu: &sync.RWMutex{},
		providerVersions:      make(map[string]*version.Version, 0),
		providerVersionsMu:    &sync.RWMutex{},
		tfVersionMu:           &sync.RWMutex{},
		tfVersionStateMu:      &sync.RWMutex{},
		tfExecPathMu:          &sync.RWMutex{},
		parsedFiles:           make(map[string]*hcl.File, 0),
		parserMu:              &sync.RWMutex{},
		parserStateMu:         &sync.RWMutex{},
		diagsMu:               &sync.RWMutex{},
	}
}

func NewModule(fs filesystem.Filesystem, dir string) Module {
	return newModule(fs, dir)
}

func (m *module) HasOpenFiles() bool {
	openFiles, err := m.fs.HasOpenFiles(m.Path())
	if err != nil {
		m.logger.Printf("%s: failed to check whether module has open files: %s",
			m.Path(), err)
	}
	return openFiles
}

func (m *module) SetTerraformVersion(v *version.Version, err error) {
	m.tfVersionMu.Lock()
	defer m.tfVersionMu.Unlock()
	m.tfVersion = v
	m.tfVersionErr = err
}

func (m *module) TerraformVersion() (*version.Version, error) {
	m.tfVersionMu.RLock()
	defer m.tfVersionMu.RUnlock()
	return m.tfVersion, m.tfVersionErr
}

func (m *module) SetProviderVersions(pv map[string]*version.Version) {
	m.providerVersionsMu.Lock()
	defer m.providerVersionsMu.Unlock()
	m.providerVersions = pv
}

func (m *module) ProviderVersions() map[string]*version.Version {
	m.providerVersionsMu.RLock()
	defer m.providerVersionsMu.RUnlock()
	return m.providerVersions
}

func (m *module) TerraformVersionState() OpState {
	m.tfVersionMu.RLock()
	defer m.tfVersionMu.RUnlock()
	return m.tfVersionState
}

func (m *module) SetTerraformVersionState(state OpState) {
	m.tfVersionMu.Lock()
	defer m.tfVersionMu.Unlock()
	m.tfVersionState = state
}

func (m *module) SetModuleManifest(manifest *datadir.ModuleManifest, err error) {
	m.modManifestMu.Lock()
	defer m.modManifestMu.Unlock()
	m.modManifest = manifest
	m.modManifestErr = err
}

func (m *module) ModuleManifestState() OpState {
	m.modManifestMu.RLock()
	defer m.modManifestMu.RUnlock()
	return m.modManigestState
}

func (m *module) SetModuleManifestParsingState(state OpState) {
	m.modManifestMu.Lock()
	defer m.modManifestMu.Unlock()
	m.modManigestState = state
}

func (m *module) SetProviderSchemas(ps *tfjson.ProviderSchemas, err error) {
	m.providerSchemaMu.Lock()
	defer m.providerSchemaMu.Unlock()
	m.providerSchema = ps
	m.providerSchemaErr = err
}

func (m *module) ProviderSchema() (*tfjson.ProviderSchemas, error) {
	m.providerSchemaMu.RLock()
	defer m.providerSchemaMu.RUnlock()
	return m.providerSchema, m.providerSchemaErr
}

func (m *module) ProviderSchemaState() OpState {
	m.providerSchemaMu.RLock()
	defer m.providerSchemaMu.RUnlock()
	return m.providerSchemaState
}

func (m *module) SetProviderSchemaObtainingState(state OpState) {
	m.providerSchemaMu.Lock()
	defer m.providerSchemaMu.Unlock()
	m.providerSchemaState = state
}

func (m *module) ParsedFiles() (map[string]*hcl.File, error) {
	m.parserMu.RLock()
	defer m.parserMu.RUnlock()
	return m.parsedFiles, m.parsingErr
}

func (m *module) SetParsedFiles(files map[string]*hcl.File, err error) {
	m.parserMu.Lock()
	defer m.parserMu.Unlock()
	m.parsedFiles = files
	m.parsingErr = err
}

func (m *module) SetDiagnostics(diags map[string]hcl.Diagnostics) {
	m.diagsMu.Lock()
	defer m.diagsMu.Unlock()
	m.diags = diags
}

func (m *module) ConfigParsingState() OpState {
	m.parserMu.RLock()
	defer m.parserMu.RUnlock()
	return m.parserState
}

func (m *module) SetConfigParsingState(state OpState) {
	m.parserMu.Lock()
	defer m.parserMu.Unlock()
	m.parserState = state
}

func (m *module) ModuleManifest() (*datadir.ModuleManifest, error) {
	m.modManifestMu.RLock()
	defer m.modManifestMu.RUnlock()
	return m.modManifest, m.modManifestErr
}

func (m *module) ModuleCalls() []datadir.ModuleRecord {
	m.modManifestMu.RLock()
	defer m.modManifestMu.RUnlock()
	if m.modManifest == nil {
		return []datadir.ModuleRecord{}
	}
	return m.modManifest.Records
}

func (m *module) CallsModule(path string) bool {
	m.modManifestMu.RLock()
	defer m.modManifestMu.RUnlock()
	if m.modManifest == nil {
		return false
	}

	for _, mod := range m.modManifest.Records {
		if mod.IsRoot() {
			// skip root module, as that's tracked separately
			continue
		}
		if mod.IsExternal() {
			// skip external modules as these shouldn't be modified from cache
			continue
		}
		absPath := filepath.Join(m.modManifest.RootDir(), mod.Dir)
		if pathcmp.PathEquals(absPath, path) {
			return true
		}
	}

	return false
}

func (m *module) SetLogger(logger *log.Logger) {
	m.logger = logger
}

func (m *module) Path() string {
	return m.path
}

func (m *module) MatchesPath(path string) bool {
	return pathcmp.PathEquals(m.path, path)
}

// HumanReadablePath helps display shorter, but still relevant paths
func (m *module) HumanReadablePath(rootDir string) string {
	if rootDir == "" {
		return m.path
	}

	// absolute paths can be too long for UI/messages,
	// so we just display relative to root dir
	relDir, err := filepath.Rel(rootDir, m.path)
	if err != nil {
		return m.path
	}

	if relDir == "." {
		// Name of the root dir is more helpful than "."
		return filepath.Base(rootDir)
	}

	return relDir
}

func (m *module) TerraformExecPath() string {
	m.tfExecPathMu.RLock()
	defer m.tfExecPathMu.RUnlock()
	return m.tfExecPath
}

func (m *module) Diagnostics() map[string]hcl.Diagnostics {
	m.diagsMu.RLock()
	defer m.diagsMu.RUnlock()
	return m.diags
}
