package schemas

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

//go:embed data
var FS embed.FS

type ProviderSchema struct {
	File    fs.File
	Version *version.Version
}

type SchemaNotAvailable struct {
	Addr tfaddr.Provider
}

func (e SchemaNotAvailable) Error() string {
	return fmt.Sprintf("embedded schema not available for %s", e.Addr)
}

func FindProviderSchemaFile(filesystem fs.ReadDirFS, pAddr tfaddr.Provider) (*ProviderSchema, error) {
	providerPath := path.Join("data", pAddr.Hostname.String(), pAddr.Namespace, pAddr.Type)

	entries, err := fs.ReadDir(filesystem, providerPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, SchemaNotAvailable{Addr: pAddr}
		}
		return nil, err
	}

	if len(entries) != 1 {
		return nil, fmt.Errorf("%q: schema not found", pAddr)
	}

	rawVersion := entries[0].Name()

	filePath := path.Join(providerPath, rawVersion, "schema.json")
	file, err := filesystem.Open(filePath)
	if err != nil {
		return nil, err
	}

	version, err := version.NewVersion(rawVersion)
	if err != nil {
		return nil, err
	}

	return &ProviderSchema{
		File:    file,
		Version: version,
	}, nil
}
