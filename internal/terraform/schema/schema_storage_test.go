package schema

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	tferr "github.com/hashicorp/terraform-ls/internal/terraform/errors"
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
			err := SchemaSupportsTerraform(tc.version)
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

func TestProviderConfigSchema_noSchema(t *testing.T) {
	s := NewStorage()
	expectedErr := &NoSchemaAvailableErr{}
	_, err := s.ProviderConfigSchema("any")
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestResourceSchema_noSchema(t *testing.T) {
	s := NewStorage()
	expectedErr := &NoSchemaAvailableErr{}
	_, err := s.ResourceSchema("any")
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestDataSourceSchema_noSchema(t *testing.T) {
	s := NewStorage()
	expectedErr := &NoSchemaAvailableErr{}
	_, err := s.DataSourceSchema("any")
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestDataSources_noSchema(t *testing.T) {
	s := NewStorage()
	expectedErr := &NoSchemaAvailableErr{}
	_, err := s.DataSources()
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestProviders_noSchema(t *testing.T) {
	s := NewStorage()
	expectedErr := &NoSchemaAvailableErr{}
	_, err := s.Providers()
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}

func TestResources_noSchema(t *testing.T) {
	s := NewStorage()
	expectedErr := &NoSchemaAvailableErr{}
	_, err := s.Resources()
	if err == nil {
		t.Fatalf("Expected error (%q)", expectedErr.Error())
	}
	if !errors.Is(err, expectedErr) {
		diff := cmp.Diff(expectedErr, err)
		t.Fatalf("Error doesn't match: %s", diff)
	}
}
