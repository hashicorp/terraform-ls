package rootmodule

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	version "github.com/hashicorp/go-version"
)

func moduleManifestFilePath(dir string) string {
	return filepath.Join(
		dir,
		".terraform",
		"modules",
		"modules.json")
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

// moduleManifest is an internal struct used only to assist in our JSON
// serialization of manifest snapshots. It should not be used for any other
// purpose.
type moduleManifest struct {
	rootDir string
	Records []ModuleRecord `json:"Modules"`
}

func ParseModuleManifestFromFile(path string) (*moduleManifest, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	mm, err := parseModuleManifest(b)
	if err != nil {
		return nil, err
	}
	mm.rootDir = rootModuleDirFromFilePath(path)

	return mm, nil
}

func parseModuleManifest(b []byte) (*moduleManifest, error) {
	mm := moduleManifest{}
	err := json.Unmarshal(b, &mm)
	if err != nil {
		return nil, err
	}

	return &mm, nil
}
