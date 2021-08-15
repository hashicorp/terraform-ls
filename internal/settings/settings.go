package settings

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
)

type ExperimentalFeatures struct {
	ValidateOnSave bool `mapstructure:"validateOnSave"`
}

type Options struct {
	// ModulePaths describes a list of absolute paths to modules to load
	ModulePaths        []string `mapstructure:"rootModulePaths"`
	ExcludeModulePaths []string `mapstructure:"excludeModulePaths"`
	CommandPrefix      string   `mapstructure:"commandPrefix"`

	// ExperimentalFeatures encapsulates experimental features users can opt into.
	ExperimentalFeatures ExperimentalFeatures `mapstructure:"experimentalFeatures"`

	TerraformExecPath    string `mapstructure:"terraformExecPath"`
	TerraformExecTimeout string `mapstructure:"terraformExecTimeout"`
	TerraformLogFilePath string `mapstructure:"terraformLogFilePath"`
}

func (o *Options) Validate() error {
	if len(o.ModulePaths) != 0 && len(o.ExcludeModulePaths) != 0 {
		return fmt.Errorf("at most one of `rootModulePaths` and `excludeModulePaths` could be set")
	}

	if o.TerraformExecPath != "" {
		path := o.TerraformExecPath
		if !filepath.IsAbs(path) {
			return fmt.Errorf("Expected absolute path for Terraform binary, got %q", path)
		}
		stat, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("Unable to find Terraform binary: %s", err)
		}
		if stat.IsDir() {
			return fmt.Errorf("Expected a Terraform binary, got a directory: %q", path)
		}
	}

	return nil
}

type DecodedOptions struct {
	Options    *Options
	UnusedKeys []string
}

func DecodeOptions(input interface{}) (*DecodedOptions, error) {
	var md mapstructure.Metadata
	var options Options

	config := &mapstructure.DecoderConfig{
		Metadata: &md,
		Result:   &options,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		panic(err)
	}

	if err := decoder.Decode(input); err != nil {
		return nil, err
	}

	return &DecodedOptions{
		Options:    &options,
		UnusedKeys: md.Unused,
	}, nil
}
