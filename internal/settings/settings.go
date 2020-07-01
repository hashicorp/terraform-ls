package settings

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/mapstructure"
)

type Options struct {
	// RootModulePaths describes a list of absolute paths to root modules
	RootModulePaths []string `mapstructure:"rootModulePaths"`

	// TODO: Need to check for conflict with CLI flags
	// TerraformExecPath string
	// TerraformExecTimeout time.Duration
	// TerraformLogFilePath string
}

func (o *Options) Validate() error {
	var result *multierror.Error

	for _, p := range o.RootModulePaths {
		if !filepath.IsAbs(p) {
			result = multierror.Append(result, fmt.Errorf("%q is not an absolute path", p))
		}
	}

	return result.ErrorOrNil()
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
