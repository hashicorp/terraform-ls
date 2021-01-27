package module

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/schemas"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	tfschema "github.com/hashicorp/terraform-schema/schema"
)

type module struct {
	path   string
	logger *log.Logger

	// loading
	isLoading     bool
	isLoadingMu   *sync.RWMutex
	loadingDone   <-chan struct{}
	cancelLoading context.CancelFunc
	loadErr       error
	loadErrMu     *sync.RWMutex

	// module cache
	moduleMu           *sync.RWMutex
	moduleManifestFile File
	moduleManifest     *moduleManifest

	// plugin (provider schema) cache
	pluginMu         *sync.RWMutex
	pluginLockFile   File
	providerSchema   *tfjson.ProviderSchemas
	providerSchemaMu *sync.RWMutex
	providerVersions map[string]*version.Version

	// terraform executor
	tfLoadingDone bool
	tfLoadingMu   *sync.RWMutex
	tfExec        exec.TerraformExecutor
	tfNewExecutor exec.ExecutorFactory
	tfExecPath    string
	tfExecTimeout time.Duration
	tfExecLogPath string

	// terraform discovery
	tfDiscoFunc  discovery.DiscoveryFunc
	tfDiscoErr   error
	tfVersion    *version.Version
	tfVersionErr error

	// core schema
	coreSchema   *schema.BodySchema
	coreSchemaMu *sync.RWMutex

	// decoder
	isParsed    bool
	isParsedMu  *sync.RWMutex
	pFilesMap   map[string]*hcl.File
	parsedDiags map[string]hcl.Diagnostics
	parserMu    *sync.RWMutex
	filesystem  filesystem.Filesystem
}

func newModule(fs filesystem.Filesystem, dir string) *module {
	return &module{
		path:             dir,
		filesystem:       fs,
		logger:           defaultLogger,
		isLoadingMu:      &sync.RWMutex{},
		loadErrMu:        &sync.RWMutex{},
		moduleMu:         &sync.RWMutex{},
		pluginMu:         &sync.RWMutex{},
		providerSchemaMu: &sync.RWMutex{},
		tfLoadingMu:      &sync.RWMutex{},
		coreSchema:       tfschema.UniversalCoreModuleSchema(),
		coreSchemaMu:     &sync.RWMutex{},
		isParsedMu:       &sync.RWMutex{},
		pFilesMap:        make(map[string]*hcl.File, 0),
		providerVersions: make(map[string]*version.Version, 0),
		parserMu:         &sync.RWMutex{},
	}
}

var defaultLogger = log.New(ioutil.Discard, "", 0)

func NewModule(ctx context.Context, fs filesystem.Filesystem, dir string) (Module, error) {
	m := newModule(fs, dir)

	d := &discovery.Discovery{}
	m.tfDiscoFunc = d.LookPath

	m.tfNewExecutor = exec.NewExecutor

	err := m.discoverCaches(ctx, dir)
	if err != nil {
		return m, err
	}

	return m, m.load(ctx)
}

func (m *module) discoverCaches(ctx context.Context, dir string) error {
	var errs *multierror.Error
	err := m.discoverPluginCache(dir)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	err = m.discoverModuleCache(dir)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	return errs.ErrorOrNil()
}

func (m *module) WasInitialized() (bool, error) {
	tfDirPath := filepath.Join(m.Path(), ".terraform")

	f, err := m.filesystem.Open(tfDirPath)
	if err != nil {
		return false, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	if !fi.IsDir() {
		return false, fmt.Errorf("%s is not a directory", tfDirPath)
	}

	return true, nil
}

func (m *module) discoverPluginCache(dir string) error {
	m.pluginMu.Lock()
	defer m.pluginMu.Unlock()

	lockPaths := pluginLockFilePaths(dir)
	lf, err := findFile(lockPaths)
	if err != nil {
		if os.IsNotExist(err) {
			m.logger.Printf("no plugin cache found: %s", err.Error())
			return nil
		}

		return fmt.Errorf("unable to calculate hash: %w", err)
	}
	m.pluginLockFile = lf
	return nil
}

func (m *module) discoverModuleCache(dir string) error {
	m.moduleMu.Lock()
	defer m.moduleMu.Unlock()

	lf, err := newFile(moduleManifestFilePath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			m.logger.Printf("no module manifest file found: %s", err.Error())
			return nil
		}

		return fmt.Errorf("unable to calculate hash: %w", err)
	}
	m.moduleManifestFile = lf
	return nil
}

func (m *module) Modules() []ModuleRecord {
	m.moduleMu.Lock()
	defer m.moduleMu.Unlock()
	if m.moduleManifest == nil {
		return []ModuleRecord{}
	}

	return m.moduleManifest.Records
}

func (m *module) SetLogger(logger *log.Logger) {
	m.logger = logger
}

func (m *module) StartLoading() error {
	if !m.IsLoadingDone() {
		return fmt.Errorf("module is already being loaded")
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	m.cancelLoading = cancelFunc
	m.loadingDone = ctx.Done()

	go func(ctx context.Context) {
		m.setLoadErr(m.load(ctx))
	}(ctx)
	return nil
}

func (m *module) CancelLoading() {
	if !m.IsLoadingDone() && m.cancelLoading != nil {
		m.cancelLoading()
	}
	m.setLoadingState(false)
}

func (m *module) LoadingDone() <-chan struct{} {
	return m.loadingDone
}

func (m *module) load(ctx context.Context) error {
	var errs *multierror.Error
	defer m.CancelLoading()

	// reset internal loading state
	m.setLoadingState(true)

	// The following operations have to happen in a particular order
	// as they depend on the internal state as mutated by each operation

	err := m.UpdateModuleManifest(m.moduleManifestFile)
	errs = multierror.Append(errs, err)

	err = m.discoverTerraformExecutor(ctx)
	m.tfDiscoErr = err
	errs = multierror.Append(errs, err)

	err = m.discoverTerraformVersion(ctx)
	m.tfVersionErr = err
	errs = multierror.Append(errs, err)

	err = m.findAndSetCoreSchema()
	if err != nil {
		m.logger.Printf("%s: %s - falling back to universal schema",
			m.Path(), err)
	}

	err = m.UpdateProviderSchemaCache(ctx, m.pluginLockFile)
	errs = multierror.Append(errs, err)

	m.logger.Printf("loading of module %s finished: %s",
		m.Path(), errs)
	return errs.ErrorOrNil()
}

func (m *module) setLoadingState(isLoading bool) {
	m.isLoadingMu.Lock()
	defer m.isLoadingMu.Unlock()
	m.isLoading = isLoading
}

func (m *module) IsLoadingDone() bool {
	m.isLoadingMu.RLock()
	defer m.isLoadingMu.RUnlock()
	return !m.isLoading
}

func (m *module) discoverTerraformExecutor(ctx context.Context) error {
	defer func() {
		m.setTfDiscoveryFinished(true)
	}()

	tfPath := m.tfExecPath
	if tfPath == "" {
		var err error
		tfPath, err = m.tfDiscoFunc()
		if err != nil {
			return err
		}
	}

	tf, err := m.tfNewExecutor(m.path, tfPath)
	if err != nil {
		return err
	}

	tf.SetLogger(m.logger)

	if m.tfExecLogPath != "" {
		tf.SetExecLogPath(m.tfExecLogPath)
	}

	if m.tfExecTimeout != 0 {
		tf.SetTimeout(m.tfExecTimeout)
	}

	m.tfExec = tf

	return nil
}

func (m *module) ExecuteTerraformInit(ctx context.Context) error {
	if !m.IsTerraformAvailable() {
		if err := m.discoverTerraformExecutor(ctx); err != nil {
			return err
		}
	}

	return m.tfExec.Init(ctx)
}

func (m *module) ExecuteTerraformPlan(ctx context.Context) error {
	if !m.IsTerraformAvailable() {
		if err := m.discoverTerraformExecutor(ctx); err != nil {
			return err
		}
	}

	return m.tfExec.Plan(ctx)
}

func (m *module) ExecuteTerraformValidate(ctx context.Context) (map[string]hcl.Diagnostics, error) {
	diagsMap := make(map[string]hcl.Diagnostics)

	if !m.IsTerraformAvailable() {
		if err := m.discoverTerraformExecutor(ctx); err != nil {
			return diagsMap, err
		}
	}

	if !m.IsParsed() {
		if err := m.ParseFiles(); err != nil {
			return diagsMap, err
		}
	}

	// an entry for each file should exist, even if there are no diags
	for filename := range m.parsedFiles() {
		diagsMap[filename] = make(hcl.Diagnostics, 0)
	}
	// since validation applies to linked modules, create an entry for all
	// files of linked modules
	for _, mod := range m.moduleManifest.Records {
		if mod.IsRoot() {
			// skip root module
			continue
		}
		if mod.IsExternal() {
			// skip external module
			continue
		}

		absPath := filepath.Join(m.moduleManifest.rootDir, mod.Dir)
		infos, err := m.filesystem.ReadDir(absPath)
		if err != nil {
			return diagsMap, fmt.Errorf("failed to read module at %q: %w", absPath, err)
		}

		for _, info := range infos {
			if info.IsDir() {
				// We only care about files
				continue
			}

			name := info.Name()
			if !strings.HasSuffix(name, ".tf") || IsIgnoredFile(name) {
				continue
			}

			// map entries are relative to the parent module path
			filename := filepath.Join(mod.Dir, name)

			diagsMap[filename] = make(hcl.Diagnostics, 0)
		}
	}

	validationDiags, err := m.tfExec.Validate(ctx)
	if err != nil {
		return diagsMap, err
	}

	// tfjson.Diagnostic is a conversion of an internal diag to terraform core,
	// tfdiags, which is effectively based on hcl.Diagnostic.
	// This process is really just converting it back to hcl.Diagnotic
	// since it is the defacto diagnostic type for our codebase currently
	// https://github.com/hashicorp/terraform/blob/ae025248cc0712bf53c675dc2fe77af4276dd5cc/command/validate.go#L138
	for _, d := range validationDiags {
		// the diagnostic must be tied to a file to exist in the map
		if d.Range == nil || d.Range.Filename == "" {
			continue
		}

		diags := diagsMap[d.Range.Filename]

		var severity hcl.DiagnosticSeverity
		if d.Severity == "error" {
			severity = hcl.DiagError
		} else if d.Severity == "warning" {
			severity = hcl.DiagWarning
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: severity,
			Summary:  d.Summary,
			Detail:   d.Detail,
			Subject: &hcl.Range{
				Filename: d.Range.Filename,
				Start:    hcl.Pos(d.Range.Start),
				End:      hcl.Pos(d.Range.End),
			},
		})
		diagsMap[d.Range.Filename] = diags
	}

	return diagsMap, nil
}

func (m *module) discoverTerraformVersion(ctx context.Context) error {
	if m.tfExec == nil {
		return errors.New("no terraform executor - unable to read version")
	}

	version, providerVersions, err := m.tfExec.Version(ctx)
	if err != nil {
		return err
	}
	m.logger.Printf("Terraform version %s found at %s for %s", version,
		m.tfExec.GetExecPath(), m.Path())
	m.tfVersion = version

	m.providerVersions = providerVersions

	return nil
}

func (m *module) findAndSetCoreSchema() error {
	if m.tfVersion == nil {
		return errors.New("unable to find core schema without version")
	}

	coreSchema, err := tfschema.CoreModuleSchemaForVersion(m.tfVersion)
	if err != nil {
		return err
	}

	m.coreSchemaMu.Lock()
	m.coreSchema = coreSchema
	m.coreSchemaMu.Unlock()

	return nil
}

func (m *module) LoadError() error {
	m.loadErrMu.RLock()
	defer m.loadErrMu.RUnlock()
	return m.loadErr
}

func (m *module) setLoadErr(err error) {
	m.loadErrMu.Lock()
	defer m.loadErrMu.Unlock()
	m.loadErr = err
}

func (m *module) Path() string {
	return m.path
}

func (m *module) MatchesPath(path string) bool {
	return filepath.Clean(m.path) == filepath.Clean(path)
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

func (m *module) UpdateModuleManifest(lockFile File) error {
	m.moduleMu.Lock()
	defer m.moduleMu.Unlock()

	if lockFile == nil {
		m.logger.Printf("ignoring module update as no lock file was found for %s", m.Path())
		return nil
	}

	m.moduleManifestFile = lockFile

	mm, err := ParseModuleManifestFromFile(lockFile.Path())
	if err != nil {
		return fmt.Errorf("failed to update module manifest: %w", err)
	}

	m.moduleManifest = mm
	m.logger.Printf("updated module manifest - %d references parsed for %s",
		len(mm.Records), m.Path())
	return nil
}

func (m *module) DecoderWithSchema(schema *schema.BodySchema) (*decoder.Decoder, error) {
	d, err := m.Decoder()
	if err != nil {
		return nil, err
	}

	d.SetSchema(schema)

	return d, nil
}

func (m *module) Decoder() (*decoder.Decoder, error) {
	d := decoder.NewDecoder()

	for name, f := range m.parsedFiles() {
		err := d.LoadFile(name, f)
		if err != nil {
			return nil, fmt.Errorf("failed to load a file: %w", err)
		}
	}
	return d, nil
}

func (m *module) IsProviderSchemaLoaded() bool {
	m.providerSchemaMu.RLock()
	defer m.providerSchemaMu.RUnlock()
	return m.providerSchema != nil
}

func (m *module) IsParsed() bool {
	m.isParsedMu.RLock()
	defer m.isParsedMu.RUnlock()
	return m.isParsed
}

func (m *module) setIsParsed(parsed bool) {
	m.isParsedMu.Lock()
	defer m.isParsedMu.Unlock()
	m.isParsed = parsed
}

func (m *module) ParseFiles() error {
	m.parserMu.Lock()
	defer m.parserMu.Unlock()

	files := make(map[string]*hcl.File, 0)
	diags := make(map[string]hcl.Diagnostics, 0)

	infos, err := m.filesystem.ReadDir(m.Path())
	if err != nil {
		return fmt.Errorf("failed to read module at %q: %w", m.Path(), err)
	}

	for _, info := range infos {
		if info.IsDir() {
			// We only care about files
			continue
		}

		name := info.Name()
		if !strings.HasSuffix(name, ".tf") || IsIgnoredFile(name) {
			continue
		}

		// TODO: overrides

		fullPath := filepath.Join(m.Path(), name)

		src, err := m.filesystem.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read %q: %s", name, err)
		}

		m.logger.Printf("parsing file %q", name)
		f, pDiags := hclsyntax.ParseConfig(src, name, hcl.InitialPos)
		diags[name] = pDiags
		if f != nil {
			files[name] = f
		}
	}

	m.pFilesMap = files
	m.parsedDiags = diags
	m.setIsParsed(true)

	return nil
}

func (m *module) ParsedDiagnostics() map[string]hcl.Diagnostics {
	m.parserMu.Lock()
	defer m.parserMu.Unlock()
	return m.parsedDiags
}

func (m *module) parsedFiles() map[string]*hcl.File {
	m.parserMu.RLock()
	defer m.parserMu.RUnlock()

	return m.pFilesMap
}

func (m *module) MergedSchema() (*schema.BodySchema, error) {
	m.coreSchemaMu.RLock()
	defer m.coreSchemaMu.RUnlock()

	if !m.IsParsed() {
		err := m.ParseFiles()
		if err != nil {
			return nil, err
		}
	}

	ps, vOut, err := schemas.PreloadedProviderSchemas()
	if err != nil {
		return nil, err
	}
	providerVersions := vOut.Providers
	tfVersion := vOut.Core

	if m.IsProviderSchemaLoaded() {
		m.providerSchemaMu.RLock()
		defer m.providerSchemaMu.RUnlock()
		ps = m.providerSchema
		providerVersions = m.providerVersions
		tfVersion = m.tfVersion
	}

	if ps == nil {
		m.logger.Print("provider schemas is nil... skipping merge with core schema")
		return m.coreSchema, nil
	}

	sm := tfschema.NewSchemaMerger(m.coreSchema)
	sm.SetCoreVersion(tfVersion)
	sm.SetParsedFiles(m.parsedFiles())

	err = sm.SetProviderVersions(providerVersions)
	if err != nil {
		return nil, err
	}

	return sm.MergeWithJsonProviderSchemas(ps)
}

// IsIgnoredFile returns true if the given filename (which must not have a
// directory path ahead of it) should be ignored as e.g. an editor swap file.
func IsIgnoredFile(name string) bool {
	return strings.HasPrefix(name, ".") || // Unix-like hidden files
		strings.HasSuffix(name, "~") || // vim
		strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#") // emacs
}

func (m *module) ReferencesModulePath(path string) bool {
	m.moduleMu.Lock()
	defer m.moduleMu.Unlock()
	if m.moduleManifest == nil {
		return false
	}

	for _, mod := range m.moduleManifest.Records {
		if mod.IsRoot() {
			// skip root module, as that's tracked separately
			continue
		}
		if mod.IsExternal() {
			// skip external modules as these shouldn't be modified from cache
			continue
		}
		absPath := filepath.Join(m.moduleManifest.rootDir, mod.Dir)
		if pathEquals(absPath, path) {
			return true
		}
	}

	return false
}

func (m *module) TerraformFormatter() (exec.Formatter, error) {
	if !m.HasTerraformDiscoveryFinished() {
		return nil, fmt.Errorf("terraform is not loaded yet")
	}

	if !m.IsTerraformAvailable() {
		return nil, fmt.Errorf("terraform is not available")
	}

	return m.tfExec.Format, nil
}

func (m *module) HasTerraformDiscoveryFinished() bool {
	m.tfLoadingMu.RLock()
	defer m.tfLoadingMu.RUnlock()
	return m.tfLoadingDone
}

func (m *module) setTfDiscoveryFinished(isLoaded bool) {
	m.tfLoadingMu.Lock()
	defer m.tfLoadingMu.Unlock()
	m.tfLoadingDone = isLoaded
}

func (m *module) IsTerraformAvailable() bool {
	return m.HasTerraformDiscoveryFinished() && m.tfExec != nil
}

func (m *module) UpdateProviderSchemaCache(ctx context.Context, lockFile File) error {
	m.pluginMu.Lock()
	defer m.pluginMu.Unlock()

	if !m.IsTerraformAvailable() {
		return fmt.Errorf("cannot update provider schema as terraform is unavailable")
	}

	if lockFile == nil {
		m.logger.Printf("ignoring provider schema update as no lock file was provided for %s",
			m.Path())
		return nil
	}

	m.pluginLockFile = lockFile

	schemas, err := m.tfExec.ProviderSchemas(ctx)
	if err != nil {
		return err
	}

	m.providerSchemaMu.Lock()
	m.providerSchema = schemas
	m.providerSchemaMu.Unlock()

	return nil
}

func (m *module) PathsToWatch() []string {
	m.pluginMu.RLock()
	m.moduleMu.RLock()
	defer m.moduleMu.RUnlock()
	defer m.pluginMu.RUnlock()

	files := make([]string, 0)
	if m.pluginLockFile != nil {
		files = append(files, m.pluginLockFile.Path())
	}
	if m.moduleManifestFile != nil {
		files = append(files, m.moduleManifestFile.Path())
	}

	return files
}

func (m *module) IsKnownModuleManifestFile(path string) bool {
	m.moduleMu.RLock()
	defer m.moduleMu.RUnlock()

	if m.moduleManifestFile == nil {
		return false
	}

	return pathEquals(m.moduleManifestFile.Path(), path)
}

func (m *module) IsKnownPluginLockFile(path string) bool {
	m.pluginMu.RLock()
	defer m.pluginMu.RUnlock()

	if m.pluginLockFile == nil {
		return false
	}

	return pathEquals(m.pluginLockFile.Path(), path)
}
