package main

import (
	"os"

	"github.com/hashicorp/terraform-ls/commands"
	"github.com/mitchellh/cli"
)

func main() {
	c := &cli.CLI{
		Name:    "terraform-ls",
		Version: "0.1.0",
		Args:    os.Args[1:],
	}

	ui := &cli.ColoredUi{
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Writer:      os.Stdout,
			Reader:      os.Stdin,
			ErrorWriter: os.Stderr,
		},
	}

	c.Commands = map[string]cli.CommandFactory{
		"completion": func() (cli.Command, error) {
			return &commands.CompletionCommand{
				Ui: ui,
			}, nil
		},
		"serve": func() (cli.Command, error) {
			return &commands.ServeCommand{
				Ui: ui,
			}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		ui.Error("Error: " + err.Error())
	}

	os.Exit(exitStatus)
}
