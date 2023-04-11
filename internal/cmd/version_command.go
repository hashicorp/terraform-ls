// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"runtime"
	"strings"

	"github.com/mitchellh/cli"
)

type VersionOutput struct {
	Version string `json:"version"`

	*BuildInfo
}

type VersionCommand struct {
	Ui      cli.Ui
	Version string

	jsonOutput bool
}

type BuildInfo struct {
	GoVersion string `json:"go,omitempty"`
	GoOS      string `json:"os,omitempty"`
	GoArch    string `json:"arch,omitempty"`
	Compiler  string `json:"compiler,omitempty"`
}

func (c *VersionCommand) flags() *flag.FlagSet {
	fs := defaultFlagSet("version")

	fs.BoolVar(&c.jsonOutput, "json", false, "output the version information as a JSON object")

	fs.Usage = func() { c.Ui.Error(c.Help()) }

	return fs
}

func (c *VersionCommand) Run(args []string) int {
	f := c.flags()
	if err := f.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s", err))
		return 1
	}

	output := VersionOutput{
		Version: c.Version,
		BuildInfo: &BuildInfo{
			GoVersion: runtime.Version(),
			GoOS:      runtime.GOOS,
			GoArch:    runtime.GOARCH,
			Compiler:  runtime.Compiler,
		},
	}

	if c.jsonOutput {
		jsonOutput, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			c.Ui.Error(fmt.Sprintf("\nError marshalling JSON: %s", err))
			return 1
		}
		c.Ui.Output(string(jsonOutput))
		return 0
	}

	ver := fmt.Sprintf("%s\nplatform: %s/%s\ngo: %s\ncompiler: %s", c.Version, output.GoOS, output.GoArch, output.GoVersion, output.Compiler)
	c.Ui.Output(ver)
	return 0
}

func (c *VersionCommand) Help() string {
	helpText := `
Usage: terraform-ls version [-json]

` + c.Synopsis() + "\n\n" + helpForFlags(c.flags())

	return strings.TrimSpace(helpText)
}

func (c *VersionCommand) Synopsis() string {
	return "Displays the version of the language server"
}
