package datadir

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/pathcmp"
)

func ModuleManifestFilePath(fs filesystem.Filesystem, modulePath string) (string, bool) {
	manifestPath := filepath.Join(
		append([]string{modulePath},
			manifestPathElements...)...)

	fi, err := fs.Stat(manifestPath)
	if err == nil && fi.Mode().IsRegular() {
		return manifestPath, true
	}
	return "", false
}

// The following structs were copied from terraform's
// internal/modsdir/manifest.go

// ModuleRecord represents some metadata about an installed module, as part
// of a ModuleManifest.
type ModuleRecord struct {
	// Key is a unique identifier for this particular module, based on its
	// position within the static module tree.
	Key string `json:"Key"`

	// SourceAddr is the source address given for this module in configuration.
	// This is used only to detect if the source was changed in configuration
	// since the module was last installed, which means that the installer
	// must re-install it.
	SourceAddr string `json:"Source"`

	// Version is the exact version of the module, which results from parsing
	// VersionStr. nil for un-versioned modules.
	Version *version.Version `json:"-"`

	// VersionStr is the version specifier string. This is used only for
	// serialization in snapshots and should not be accessed or updated
	// by any other codepaths; use "Version" instead.
	VersionStr string `json:"Version,omitempty"`

	// Dir is the path to the local directory where the module is installed.
	Dir string `json:"Dir"`
}

func (r *ModuleRecord) UnmarshalJSON(b []byte) error {
	type rawRecord ModuleRecord
	var record rawRecord

	err := json.Unmarshal(b, &record)
	if err != nil {
		return err
	}
	if record.VersionStr != "" {
		record.Version, err = version.NewVersion(record.VersionStr)
		if err != nil {
			return fmt.Errorf("invalid version %q for %s: %s", record.VersionStr, record.Key, err)
		}
	}

	// Ensure Windows is using the proper modules path format after
	// reading the modules manifest Dir records
	record.Dir = filepath.FromSlash(record.Dir)

	// Terraform should be persisting clean paths already
	// but it doesn't hurt to clean them for sanity
	record.Dir = filepath.Clean(record.Dir)

	// TODO: Follow symlinks (requires proper test data)

	*r = (ModuleRecord)(record)

	return nil
}

func (r *ModuleRecord) IsRoot() bool {
	return r.Key == ""
}

func (r *ModuleRecord) IsExternal() bool {
	modCacheDir := filepath.Join(".terraform", "modules")
	if strings.HasPrefix(r.Dir, modCacheDir) {
		return true
	}

	return false
}

type ModuleManifest struct {
	rootDir string
	Records []ModuleRecord `json:"Modules"`
}

func (mm *ModuleManifest) Copy() *ModuleManifest {
	if mm == nil {
		return nil
	}

	newMm := &ModuleManifest{
		rootDir: mm.rootDir,
		Records: make([]ModuleRecord, len(mm.Records)),
	}

	for i, record := range mm.Records {
		// Individual records are immutable once parsed
		newMm.Records[i] = record
	}

	return newMm
}

func NewModuleManifest(rootDir string, records []ModuleRecord) *ModuleManifest {
	return &ModuleManifest{
		rootDir: rootDir,
		Records: records,
	}
}

func (mm *ModuleManifest) RootDir() string {
	return mm.rootDir
}

func (mm *ModuleManifest) ContainsLocalModule(path string) bool {
	for _, mod := range mm.Records {
		if mod.IsRoot() || mod.IsExternal() {
			continue
		}

		absPath := filepath.Join(mm.RootDir(), mod.Dir)
		if pathcmp.PathEquals(absPath, path) {
			return true
		}
	}
	return false
}

func ParseModuleManifestFromFile(path string) (*ModuleManifest, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	mm, err := parseModuleManifest(b)
	if err != nil {
		return nil, err
	}
	rootDir, ok := ModulePath(path)
	if !ok {
		return nil, fmt.Errorf("failed to detect module path: %s", path)
	}
	mm.rootDir = filepath.Clean(rootDir)

	return mm, nil
}

func parseModuleManifest(b []byte) (*ModuleManifest, error) {
	mm := ModuleManifest{}
	err := json.Unmarshal(b, &mm)
	if err != nil {
		return nil, err
	}

	return &mm, nil
}
