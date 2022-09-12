//go:build preloadschema
// +build preloadschema

package schemas

import (
	_ "embed"
	"encoding/json"
	"sync"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
)

//go:embed data/schemas.json
var schemasJsonBytes []byte

//go:embed data/versions.json
var versionsJsonBytes []byte

var (
	_preloadedProviderSchemas     *tfjson.ProviderSchemas
	_preloadedVersionOutput       VersionOutput
	_preloadedProviderSchemasOnce sync.Once
	_preloadedProviderSchemasErr  error
)

func PreloadedProviderSchemas() (*tfjson.ProviderSchemas, VersionOutput, error) {
	_preloadedProviderSchemasOnce.Do(func() {
		_preloadedProviderSchemas = &tfjson.ProviderSchemas{}
		_preloadedProviderSchemasErr = json.Unmarshal(schemasJsonBytes, _preloadedProviderSchemas)

		output := &RawVersionOutput{}
		err := json.Unmarshal(versionsJsonBytes, output)
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

		_preloadedVersionOutput = VersionOutput{
			Core:      coreVersion,
			Providers: pVersions,
		}
	})

	return _preloadedProviderSchemas, _preloadedVersionOutput, _preloadedProviderSchemasErr
}
