// +build release

package schemas

import (
	"encoding/json"
	"sync"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
)

var (
	_preloadedProviderSchemas     *tfjson.ProviderSchemas
	_preloadedVersionOutput       Version
	_preloadedProviderSchemasOnce sync.Once
	_preloadedProviderSchemasErr  error
)

func PreloadedProviderSchemas() (*tfjson.ProviderSchemas, Version, error) {
	_preloadedProviderSchemasOnce.Do(func() {
		schemasFile, fErr := files.Open("schemas.json")
		if fErr != nil {
			_preloadedProviderSchemasErr = fErr
			return
		}

		_preloadedProviderSchemas = &tfjson.ProviderSchemas{}
		_preloadedProviderSchemasErr = json.NewDecoder(schemasFile).Decode(_preloadedProviderSchemas)

		versionFile, fErr := files.Open("versions.json")
		if fErr != nil {
			_preloadedProviderSchemasErr = fErr
			return
		}

		output := &VersionOutput{}
		err := json.NewDecoder(versionFile).Decode(output)
		if err != nil {
			_preloadedProviderSchemasErr = err
			return
		}

		coreVersion, err := version.NewVersion(output.CoreVersion)
		if err != nil {
			_preloadedProviderSchemasErr = err
			return
		}

		pVersions := make(map[string]*version.Version, 0)
		for addr, versionString := range output.Providers {
			v, err := version.NewVersion(versionString)
			if err != nil {
				_preloadedProviderSchemasErr = err
				return
			}
			pVersions[addr] = v
		}

		_preloadedVersionOutput = Version{
			Core:      coreVersion,
			Providers: pVersions,
		}
	})

	return _preloadedProviderSchemas, _preloadedVersionOutput, _preloadedProviderSchemasErr
}
