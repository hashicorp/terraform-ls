package schema

import (
	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/addrs"
)

func NewMockStorage(ps *tfjson.ProviderSchemas) StorageFactory {
	return func(v *version.Version) (*Storage, error) {
		s, err := NewStorageForVersion(v)
		if err != nil {
			return nil, err
		}
		if ps == nil {
			ps = &tfjson.ProviderSchemas{}
		}
		s.ps = ps
		return s, nil
	}
}

type MockReader struct {
	ProviderSchemas *tfjson.ProviderSchemas

	ProviderSchemaErr   error
	ProvidersErr        error
	ResourceSchemaErr   error
	ResourcesErr        error
	DataSourceSchemaErr error
	DataSourcesErr      error
}

func (r *MockReader) storage() *Storage {
	ver := version.Must(version.NewVersion("0.12.0"))
	ss, _ := NewMockStorage(r.ProviderSchemas)(ver)
	// TODO: err handling
	return ss
}

func (r *MockReader) ProviderConfigSchema(name addrs.Provider) (*tfjson.Schema, error) {
	if r.ProviderSchemaErr != nil {
		return nil, r.ProviderSchemaErr
	}
	return r.storage().ProviderConfigSchema(name)
}
func (r *MockReader) Providers() ([]addrs.Provider, error) {
	if r.ProviderSchemaErr != nil {
		return nil, r.ProviderSchemaErr
	}
	return r.storage().Providers()
}

func (r *MockReader) ResourceSchema(pAddr addrs.Provider, rType string) (*tfjson.Schema, error) {
	if r.ResourceSchemaErr != nil {
		return nil, r.ResourceSchemaErr
	}
	return r.storage().ResourceSchema(pAddr, rType)
}
func (r *MockReader) Resources() ([]Resource, error) {
	if r.ResourceSchemaErr != nil {
		return nil, r.ResourceSchemaErr
	}
	return r.storage().Resources()
}

func (r *MockReader) DataSourceSchema(pAddr addrs.Provider, dsType string) (*tfjson.Schema, error) {
	if r.DataSourceSchemaErr != nil {
		return nil, r.DataSourceSchemaErr
	}
	return r.storage().DataSourceSchema(pAddr, dsType)
}
func (r *MockReader) DataSources() ([]DataSource, error) {
	if r.DataSourceSchemaErr != nil {
		return nil, r.DataSourceSchemaErr
	}
	return r.storage().DataSources()
}
