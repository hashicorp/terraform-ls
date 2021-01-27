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
}

type VersionCommand struct {
	Ui      cli.Ui
	Version string

	jsonOutput bool
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

	if c.jsonOutput {
		output := VersionOutput{
			Version: c.Version,
		}
		jsonOutput, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			c.Ui.Error(fmt.Sprintf("\nError marshalling JSON: %s", err))
			return 1
		}
		c.Ui.Output(string(jsonOutput))
		return 0
	}

	c.Ui.Output(string(c.Version))
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
