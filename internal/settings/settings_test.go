package settings

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeOptions_nil(t *testing.T) {
	out, err := DecodeOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	opts := out.Options

	if opts.RootModulePaths != nil {
		t.Fatalf("expected no options for nil, %#v given", opts.RootModulePaths)
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
	if diff := cmp.Diff(expectedPaths, opts.RootModulePaths); diff != "" {
		t.Fatalf("options mismatch: %s", diff)
	}
}

func TestDecodedOptions_Validate(t *testing.T) {
	opts := &Options{
		RootModulePaths: []string{
			"./relative/path",
		},
	}
	err := opts.Validate()
	if err == nil {
		t.Fatal("expected relative path to fail validation")
	}
}
