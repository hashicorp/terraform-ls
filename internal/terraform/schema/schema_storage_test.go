package schema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-ls/internal/terraform/addrs"
	tferr "github.com/hashicorp/terraform-ls/internal/terraform/errors"
	"github.com/zclconf/go-cty/cty"
)

func TestSchemaSupportsTerraform(t *testing.T) {
	testCases := []struct {
		version     string
		expectedErr error
	}{
		{
			"0.11.0",
			&tferr.UnsupportedTerraformVersion{Version: "0.11.0"},
		},
		{
			"0.12.0-rc1",
			nil,
		},
		{
			"0.12.0",
			nil,
		},
		{
			"0.13.0-beta1",
			nil,
		},
		{
			"0.14.0-beta1",
			nil,
		},
		{
			"0.14.0",
			nil,
		},
		{
			"1.0.0",
			nil,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := NewStorageForVersion(testVersion(t, tc.version))
			if err != nil {
				if tc.expectedErr == nil {
					t.Fatalf("Expected no error for %q: %#v",
						tc.version, err)
				}
				if !errors.Is(err, tc.expectedErr) {
					diff := cmp.Diff(tc.expectedErr, err)
					t.Fatalf("%q: error doesn't match: %s",
						tc.version, diff)
				}
			} else if tc.expectedErr != nil {
				t.Fatalf("Expected error for %q: %#v",
					tc.version, tc.expectedErr)
			}
		})
	}
}

func TestProviderConfigSchema_basic_v012(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = s.UpdateSchemas(ctx, parseSchemas(t, "./testdata/null-schema-0.12.json"))
	if err != nil {
		t.Fatal(err)
	}

	ps, err := s.ProviderConfigSchema(addrs.ImpliedProviderForUnqualifiedType("null"))
	if err != nil {
		t.Fatal(err)
	}

	expectedSchema := &tfjson.Schema{
		Version: 0,
		Block: &tfjson.SchemaBlock{
			Attributes: nil,
		},
	}
	if diff := cmp.Diff(expectedSchema, ps); diff != "" {
		t.Fatalf("Provider schema mismatch: %s", diff)
	}
}

func TestProviders_v012(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = s.UpdateSchemas(ctx, parseSchemas(t, "./testdata/null-schema-0.12.json"))
	if err != nil {
		t.Fatal(err)
	}

	ps, err := s.Providers()
	if err != nil {
		t.Fatal(err)
	}

	expectedList := []addrs.Provider{
		{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"},
	}
	if diff := cmp.Diff(expectedList, ps); diff != "" {
		t.Fatalf("Provider schema mismatch: %s", diff)
	}
}

func TestProviderConfigSchema_basic_v013(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.13.0"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = s.UpdateSchemas(ctx, parseSchemas(t, "./testdata/null-schema-0.13.json"))
	if err != nil {
		t.Fatal(err)
	}

	ps, err := s.ProviderConfigSchema(addrs.ImpliedProviderForUnqualifiedType("null"))
	if err != nil {
		t.Fatal(err)
	}

	expectedSchema := &tfjson.Schema{
		Version: 0,
		Block: &tfjson.SchemaBlock{
			Attributes:      nil,
			DescriptionKind: "plain",
		},
	}
	if diff := cmp.Diff(expectedSchema, ps); diff != "" {
		t.Fatalf("Provider schema mismatch: %s", diff)
	}
}

func TestProviders_v013(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.13.0"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = s.UpdateSchemas(ctx, parseSchemas(t, "./testdata/null-schema-0.13.json"))
	if err != nil {
		t.Fatal(err)
	}

	ps, err := s.Providers()
	if err != nil {
		t.Fatal(err)
	}

	expectedList := []addrs.Provider{
		{Hostname: "registry.terraform.io", Namespace: "hashicorp", Type: "null"},
	}
	if diff := cmp.Diff(expectedList, ps); diff != "" {
		t.Fatalf("Provider schema mismatch: %s", diff)
	}
}

func TestProviderConfigSchema_noSchema(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	expectedErr := &NoSchemaAvailableErr{}
	_, err = s.ProviderConfigSchema(addrs.ImpliedProviderForUnqualifiedType("any"))
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestResourceSchema_basic(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = s.UpdateSchemas(ctx, parseSchemas(t, "./testdata/null-schema-0.12.json"))
	if err != nil {
		t.Fatal(err)
	}

	nullAddr := addrs.ImpliedProviderForUnqualifiedType("null")
	given, err := s.ResourceSchema(nullAddr, "null_resource")
	if err != nil {
		t.Fatal(err)
	}
	expectedSchema := &tfjson.Schema{
		Block: &tfjson.SchemaBlock{
			Attributes: map[string]*tfjson.SchemaAttribute{
				"id": {
					AttributeType: cty.String,
					Optional:      true,
					Computed:      true,
				},
				"triggers": {
					AttributeType: cty.Map(cty.String),
					Optional:      true,
				},
			},
		},
	}
	opts := cmpopts.IgnoreUnexported(cty.Type{})
	if diff := cmp.Diff(expectedSchema, given, opts); diff != "" {
		t.Fatalf("schema mismatch: %s", diff)
	}
}

func TestResourceSchema_noSchema(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	expectedErr := &NoSchemaAvailableErr{}
	anyAddr := addrs.ImpliedProviderForUnqualifiedType("any")
	_, err = s.ResourceSchema(anyAddr, "any_res")
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestDataSourceSchema_noSchema(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	expectedErr := &NoSchemaAvailableErr{}
	anyAddr := addrs.ImpliedProviderForUnqualifiedType("any")
	_, err = s.DataSourceSchema(anyAddr, "any_res")
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestDataSourceSchema_basic(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = s.UpdateSchemas(ctx, parseSchemas(t, "./testdata/null-schema-0.12.json"))
	if err != nil {
		t.Fatal(err)
	}

	nullAddr := addrs.ImpliedProviderForUnqualifiedType("null")
	given, err := s.DataSourceSchema(nullAddr, "null_data_source")
	if err != nil {
		t.Fatal(err)
	}
	expectedSchema := &tfjson.Schema{
		Block: &tfjson.SchemaBlock{
			Attributes: map[string]*tfjson.SchemaAttribute{
				"has_computed_default": {
					AttributeType: cty.String,
					Optional:      true,
					Computed:      true,
				},
				"id": {
					AttributeType: cty.String,
					Optional:      true,
					Computed:      true,
				},
				"inputs": {
					AttributeType: cty.Map(cty.String),
					Optional:      true,
				},
				"outputs": {
					AttributeType: cty.Map(cty.String),
					Computed:      true,
				},
				"random": {
					AttributeType: cty.String,
					Computed:      true,
				},
			},
		},
	}
	opts := cmpopts.IgnoreUnexported(cty.Type{})
	if diff := cmp.Diff(expectedSchema, given, opts); diff != "" {
		t.Fatalf("schema mismatch: %s", diff)
	}
}

func TestDataSources_noSchema(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	expectedErr := &NoSchemaAvailableErr{}
	_, err = s.DataSources()
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestDataSources_basic(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = s.UpdateSchemas(ctx, parseSchemas(t, "./testdata/null-schema-0.12.json"))
	if err != nil {
		t.Fatal(err)
	}

	given, err := s.DataSources()
	if err != nil {
		t.Fatal(err)
	}
	expectedDs := []DataSource{
		{
			Name: "null_data_source", Provider: addrs.Provider{
				Hostname:  "registry.terraform.io",
				Namespace: "hashicorp",
				Type:      "null",
			},
		},
	}
	if diff := cmp.Diff(expectedDs, given); diff != "" {
		t.Fatalf("data sources mismatch: %s", diff)
	}
}

func TestProviders_noSchema(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	expectedErr := &NoSchemaAvailableErr{}
	_, err = s.Providers()
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestResources_noSchema(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	expectedErr := &NoSchemaAvailableErr{}
	_, err = s.Resources()
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestResources_basic(t *testing.T) {
	s, err := NewStorageForVersion(testVersion(t, "0.12.0"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = s.UpdateSchemas(ctx, parseSchemas(t, "./testdata/null-schema-0.12.json"))
	if err != nil {
		t.Fatal(err)
	}

	given, err := s.Resources()
	if err != nil {
		t.Fatal(err)
	}
	expectedDs := []Resource{
		{
			Name: "null_resource", Provider: addrs.Provider{
				Hostname:  "registry.terraform.io",
				Namespace: "hashicorp",
				Type:      "null",
			},
		},
	}
	if diff := cmp.Diff(expectedDs, given); diff != "" {
		t.Fatalf("resources mismatch: %s", diff)
	}
}

func parseSchemas(t *testing.T, pathToSchema string) *tfjson.ProviderSchemas {
	b, err := ioutil.ReadFile(pathToSchema)
	if err != nil {
		t.Fatal(err)
	}

	var schemas tfjson.ProviderSchemas
	err = json.Unmarshal(b, &schemas)
	if err != nil {
		t.Fatal(err)
	}

	return &schemas
}

func TempDir(t *testing.T) string {
	tmpDir := filepath.Join(os.TempDir(), "terraform-ls", t.Name())

	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		if os.IsExist(err) {
			return tmpDir
		}
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
	})

	return tmpDir
}

func testVersion(t *testing.T, v string) *version.Version {
	ver, err := version.NewVersion(v)
	if err != nil {
		t.Fatal(err)
	}
	return ver
}
