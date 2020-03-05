package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/mitchellh/cli"
	lsctx "github.com/radeksimko/terraform-ls/internal/context"
	"github.com/radeksimko/terraform-ls/langserver"
)

type serveCommand struct {
	Ui     cli.Ui
	Logger *log.Logger
}

func (c *serveCommand) Run(args []string) int {
	cmdFlags := defaultFlagSet("serve")

	var port int
	cmdFlags.IntVar(&port, "port", -1, "port")

	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	ctx, cancelFunc := lsctx.WithSignalCancel(context.Background(), c.Logger,
		syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	srv := langserver.NewLangServer(ctx, c.Logger)

	if port != -1 {
		srv.StartTCP(fmt.Sprintf("localhost:%d", port))
		return 0
	}

	srv.Start(os.Stdin, os.Stdout)

	return 0
}

func (c *serveCommand) Help() string {
	helpText := `
Usage: terraform-ls serve [options] [path]

`
	return strings.TrimSpace(helpText)
}

func (c *serveCommand) Synopsis() string {
	return "Starts the Language Server"
}
