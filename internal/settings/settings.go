package settings

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

type ExperimentalFeatures struct {
	ValidateOnSave bool `mapstructure:"validateOnSave"`
}

type Options struct {
	// RootModulePaths describes a list of absolute paths to root modules
	RootModulePaths    []string `mapstructure:"rootModulePaths"`
	ExcludeModulePaths []string `mapstructure:"excludeModulePaths"`
	CommandPrefix      string   `mapstructure:"commandPrefix"`

	// ExperimentalFeatures encapsulates experimental features users can opt into.
	ExperimentalFeatures ExperimentalFeatures `mapstructure:"experimentalFeatures"`

	// TODO: Need to check for conflict with CLI flags
	// TerraformExecPath string
	// TerraformExecTimeout time.Duration
	// TerraformLogFilePath string
}

func (o *Options) Validate() error {
	if len(o.RootModulePaths) != 0 && len(o.ExcludeModulePaths) != 0 {
		return fmt.Errorf("at most one of `rootModulePaths` and `excludeModulePaths` could be set")
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
