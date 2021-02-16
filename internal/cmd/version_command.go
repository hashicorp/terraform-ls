package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/mitchellh/cli"
)

type VersionOutput struct {
	Version string `json:"version"`

	BuildGoVersion string `json:"go_version,omitempty"`
	BuildGoOS      string `json:"go_os,omitempty"`
	BuildGoArch    string `json:"go_arch,omitempty"`
}

type VersionCommand struct {
	Ui        cli.Ui
	Version   string
	BuildInfo *BuildInfo

	jsonOutput bool
}

type BuildInfo struct {
	GoVersion string
	GoOS      string
	GoArch    string
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
		Version:        c.Version,
		BuildGoVersion: c.BuildInfo.GoVersion,
		BuildGoOS:      c.BuildInfo.GoOS,
		BuildGoArch:    c.BuildInfo.GoArch,
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

	ver := string(c.Version)
	if output.BuildGoVersion != "" && output.BuildGoOS != "" && output.BuildGoArch != "" {
		ver = fmt.Sprintf("%s\ngo%s %s/%s", c.Version, output.BuildGoVersion, output.BuildGoOS, output.BuildGoArch)
	}

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
