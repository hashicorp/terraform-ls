package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

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
			return &completionCommand{Ui: ui}, nil
		},
		"serve": func() (cli.Command, error) {
			return &serveCommand{Ui: ui, Logger: logger}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}

func defaultFlagSet(cmdName string) *flag.FlagSet {
	f := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)

	// Set the default Usage to empty
	f.Usage = func() {}

	return f
}
