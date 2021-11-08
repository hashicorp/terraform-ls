package settings

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-ls/internal/terraform/datadir"
)

func TestDecodeOptions_nil(t *testing.T) {
	out, err := DecodeOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	opts := out.Options

	if opts.ModulePaths != nil {
		t.Fatalf("expected no options for nil, %#v given", opts.ModulePaths)
	}
}

func TestDecodeOptions_wrongType(t *testing.T) {
	_, err := DecodeOptions(map[string]interface{}{
		"rootModulePaths": "/random/path",
	})
	if err == nil {
		t.Fatal("expected decoding of wrong type to result in error")
	}
}

func TestDecodeOptions_success(t *testing.T) {
	out, err := DecodeOptions(map[string]interface{}{
		"rootModulePaths": []string{"/random/path"},
	})
	if err != nil {
		t.Fatal(err)
	}
	opts := out.Options
	expectedPaths := []string{"/random/path"}
	if diff := cmp.Diff(expectedPaths, opts.ModulePaths); diff != "" {
		t.Fatalf("options mismatch: %s", diff)
	}
}

func TestValidate_IgnoreDirectoryNames_error(t *testing.T) {
	tables := []struct {
		input  string
		result string
	}{
		{datadir.DataDirName, `cannot ignore directory ".terraform"`},
		{filepath.Join("path", "path"), fmt.Sprintf(`expected directory name, got a path: %q`, filepath.Join("path", "path"))},
	}

	for _, table := range tables {
		out, err := DecodeOptions(map[string]interface{}{
			"ignoreDirectoryNames": []string{table.input},
		})
		if err != nil {
			t.Fatal(err)
		}

		result := out.Options.Validate()
		if result.Error() != table.result {
			t.Fatalf("expected error: %s, got: %s", table.result, result)
		}
	}
}
func TestValidate_IgnoreDirectoryNames_success(t *testing.T) {
	out, err := DecodeOptions(map[string]interface{}{
		"ignoreDirectoryNames": []string{"directory"},
	})
	if err != nil {
		t.Fatal(err)
	}

	result := out.Options.Validate()
	if result != nil {
		t.Fatalf("did not expect error: %s", result)
	}
}
