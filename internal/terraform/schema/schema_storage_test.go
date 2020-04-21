package schema

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
