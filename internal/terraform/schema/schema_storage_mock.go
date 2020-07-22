package schema

import (
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/addrs"
)

func NewMockStorage(ps *tfjson.ProviderSchemas) StorageFactory {
	return func(v string) (*Storage, error) {
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
	ss, _ := NewMockStorage(r.ProviderSchemas)("0.12.0")
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

func (r *MockReader) ResourceSchema(rType string) (*tfjson.Schema, error) {
	if r.ResourceSchemaErr != nil {
		return nil, r.ResourceSchemaErr
	}
	return r.storage().ResourceSchema(rType)
}
func (r *MockReader) Resources() ([]Resource, error) {
	if r.ResourceSchemaErr != nil {
		return nil, r.ResourceSchemaErr
	}
	return r.storage().Resources()
}

func (r *MockReader) DataSourceSchema(dsType string) (*tfjson.Schema, error) {
	if r.DataSourceSchemaErr != nil {
		return nil, r.DataSourceSchemaErr
	}
	return r.storage().DataSourceSchema(dsType)
}
func (r *MockReader) DataSources() ([]DataSource, error) {
	if r.DataSourceSchemaErr != nil {
		return nil, r.DataSourceSchemaErr
	}
	return r.storage().DataSources()
}
