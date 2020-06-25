package commands

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/langserver"
	"github.com/hashicorp/terraform-ls/langserver/handlers"
	"github.com/hashicorp/terraform-ls/logging"
	"github.com/mitchellh/cli"
)

type ServeCommand struct {
	Ui      cli.Ui
	Version string

	// flags
	port          int
	logFilePath   string
	tfExecPath    string
	tfExecLogPath string
	tfExecTimeout string
}

func (c *ServeCommand) flags() *flag.FlagSet {
	fs := defaultFlagSet("serve")

	fs.IntVar(&c.port, "port", 0, "port number to listen on (turns server into TCP mode)")
	fs.StringVar(&c.logFilePath, "log-file", "", "path to a file to log into with support "+
		"for variables (e.g. Timestamp, Pid, Ppid) via Go template syntax {{.VarName}}")
	fs.StringVar(&c.tfExecPath, "tf-exec", "", "path to Terraform binary")
	fs.StringVar(&c.tfExecTimeout, "tf-exec-timeout", "", "Overrides Terraform execution timeout (e.g. 30s)")
	fs.StringVar(&c.tfExecLogPath, "tf-log-file", "", "path to a file for Terraform executions"+
		" to be logged into with support for variables (e.g. Timestamp, Pid, Ppid) via Go template"+
		" syntax {{.VarName}}")

	fs.Usage = func() { c.Ui.Error(c.Help()) }

	return fs
}

func (c *ServeCommand) Run(args []string) int {
	f := c.flags()
	if err := f.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s", err))
		return 1
	}

	var logger *log.Logger
	if c.logFilePath != "" {
		fl, err := logging.NewFileLogger(c.logFilePath)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to setup file logging: %s", err))
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
			c.Ui.Error(fmt.Sprintf("Failed to setup logging for Terraform: %s", err))
			return 1
		}
		ctx = lsctx.WithTerraformExecLogPath(c.tfExecLogPath, ctx)
		logger.Printf("Terraform executions will be logged to %s "+
			"(interpolated at the time of execution)", c.tfExecLogPath)
	}

	if c.tfExecTimeout != "" {
		d, err := time.ParseDuration(c.tfExecTimeout)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to parse Terraform timeout: %s", err))
			return 1
		}
		ctx = lsctx.WithTerraformExecTimeout(d, ctx)
		logger.Printf("Terraform execution timeout set to %s", d)
	}

	if c.tfExecPath != "" {
		path := c.tfExecPath

		// just some sanity checking here, no need to get too specific otherwise will be complex cross-OS
		if !filepath.IsAbs(path) {
			c.Ui.Error(fmt.Sprintf("Expected absolute path for Terraform binary, got %q", path))
			return 1
		}
		stat, err := os.Stat(path)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Unable to find Terraform binary: %s", err))
			return 1
		}
		if stat.IsDir() {
			c.Ui.Error(fmt.Sprintf("Expected a Terraform binary, got a directory: %q", path))
			return 1
		}

		ctx = lsctx.WithTerraformExecPath(path, ctx)
		logger.Printf("Terraform exec path set to %q", path)
	}

	logger.Printf("Starting terraform-ls %s", c.Version)

	srv := langserver.NewLangServer(ctx, handlers.NewSession)
	srv.SetLogger(logger)

	if c.port != 0 {
		err := srv.StartTCP(fmt.Sprintf("localhost:%d", c.port))
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to start TCP server: %s", err))
			return 1
		}
		return 0
	}

	err := srv.StartAndWait(os.Stdin, os.Stdout)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to start server: %s", err))
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
