package commands

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/handlers"
	"github.com/mitchellh/cli"
)

type ServeCommand struct {
	Ui cli.Ui

	// flags
	port    int
	logFile string
}

func (c *ServeCommand) flags() *flag.FlagSet {
	fs := defaultFlagSet("serve")

	fs.IntVar(&c.port, "port", 0, "port number to listen on (turns server into TCP mode)")
	fs.StringVar(&c.logFile, "log-file", "", "path to file to log into with support "+
		"for variables (e.g. Timestamp, Pid, Ppid) via Go template syntax {{.VarName}}")

	fs.Usage = func() { c.Ui.Error(c.Help()) }

	return fs
}

func (c *ServeCommand) Run(args []string) int {
	f := c.flags()
	if err := f.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	var logger *log.Logger
	if c.logFile != "" {
		fl, err := NewFileLogger(c.logFile)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to setup file logging: %s\n", err.Error()))
			return 1
		}
		defer fl.Close()

		logger = fl.Logger()
	} else {
		logger = NewLogger(os.Stderr)
	}

	ctx, cancelFunc := lsctx.WithSignalCancel(context.Background(), logger,
		syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	hp := handlers.New()
	srv := langserver.NewLangServer(ctx, hp)
	srv.SetLogger(logger)

	if c.port != 0 {
		srv.StartTCP(fmt.Sprintf("localhost:%d", c.port))
		return 0
	}

	srv.StartAndWait(os.Stdin, os.Stdout)

	return 0
}

func (c *ServeCommand) Help() string {
	helpText := `
Usage: terraform-ls serve [options]

` + c.Synopsis() + "\n\n" + helpForFlags(c.flags())

	return strings.TrimSpace(helpText)
}

func (c *ServeCommand) Synopsis() string {
	return "Starts the Language Server"
}
