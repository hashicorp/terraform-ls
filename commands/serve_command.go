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
	"github.com/hashicorp/terraform-ls/logging"
	"github.com/mitchellh/cli"
)

type ServeCommand struct {
	Ui cli.Ui

	// flags
	port          int
	logFilePath   string
	tfExecLogPath string
}

func (c *ServeCommand) flags() *flag.FlagSet {
	fs := defaultFlagSet("serve")

	fs.IntVar(&c.port, "port", 0, "port number to listen on (turns server into TCP mode)")
	fs.StringVar(&c.logFilePath, "log-file", "", "path to a file to log into with support "+
		"for variables (e.g. Timestamp, Pid, Ppid) via Go template syntax {{.VarName}}")
	fs.StringVar(&c.tfExecLogPath, "tf-log-file", "", "path to a file for Terraform executions"+
		" to be logged into with support for variables (e.g. Timestamp, Pid, Ppid) via Go template"+
		" syntax {{.VarName}}")

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
	if c.logFilePath != "" {
		fl, err := logging.NewFileLogger(c.logFilePath)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to setup file logging: %s\n", err.Error()))
			return 1
		}
		defer fl.Close()

		logger = fl.Logger()
	} else {
		logger = logging.NewLogger(os.Stderr)
	}

	ctx, cancelFunc := lsctx.WithSignalCancel(context.Background(), logger,
		syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	if c.tfExecLogPath != "" {
		err := logging.ValidateExecLogPath(c.tfExecLogPath)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to setup logging for Terraform: %s\n", err.Error()))
			return 1
		}
		ctx = lsctx.WithTerraformExecLogPath(c.tfExecLogPath, ctx)
		logger.Printf("Terraform executions will be logged to %s "+
			"(interpolated at the time of execution)", c.tfExecLogPath)
	}

	srv := langserver.NewLangServer(ctx, handlers.NewService)
	srv.SetLogger(logger)

	if c.port != 0 {
		err := srv.StartTCP(fmt.Sprintf("localhost:%d", c.port))
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to start TCP server: %s\n", err))
			return 1
		}
		return 0
	}

	err := srv.StartAndWait(os.Stdin, os.Stdout)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to start server: %s\n", err))
		return 1
	}

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
