package main

import (
	"log"
	"os"

	"github.com/hashicorp/terraform-ls/commands"
	"github.com/mitchellh/cli"
)

func main() {
	c := cli.NewCLI("terraform-ls", "0.1.0")
	c.Args = os.Args[1:]

	ui := &cli.ColoredUi{
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Writer:      os.Stdout,
			Reader:      os.Stdin,
			ErrorWriter: os.Stderr,
		},
	}

	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)

	c.Commands = map[string]cli.CommandFactory{
		"completion": func() (cli.Command, error) {
			return &commands.CompletionCommand{
				Ui:     ui,
				Logger: logger,
			}, nil
		},
		"serve": func() (cli.Command, error) {
			return &commands.ServeCommand{
				Ui:     ui,
				Logger: logger,
			}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
