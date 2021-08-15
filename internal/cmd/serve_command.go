package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/handlers"
	"github.com/hashicorp/terraform-ls/internal/logging"
	"github.com/hashicorp/terraform-ls/internal/pathtpl"
	"github.com/mitchellh/cli"
)

type ServeCommand struct {
	Ui      cli.Ui
	Version string

	// flags
	port           int
	logFilePath    string
	tfExecPath     string
	tfExecLogPath  string
	tfExecTimeout  string
	cpuProfile     string
	memProfile     string
	reqConcurrency int
}

func (c *ServeCommand) flags() *flag.FlagSet {
	fs := defaultFlagSet("serve")

	fs.IntVar(&c.port, "port", 0, "port number to listen on (turns server into TCP mode)")
	fs.StringVar(&c.logFilePath, "log-file", "", "path to a file to log into with support "+
		"for variables (e.g. Timestamp, Pid, Ppid) via Go template syntax {{.VarName}}")
	fs.StringVar(&c.tfExecPath, "tf-exec", "", "(DEPRECATED) path to Terraform binary. Use terraformExecPath LSP config option instead.")
	fs.StringVar(&c.tfExecTimeout, "tf-exec-timeout", "", "(DEPRECATED) Overrides Terraform execution timeout (e.g. 30s)")
	fs.StringVar(&c.tfExecLogPath, "tf-log-file", "", "(DEPRECATED) path to a file for Terraform executions"+
		" to be logged into with support for variables (e.g. Timestamp, Pid, Ppid) via Go template"+
		" syntax {{.VarName}}")
	fs.StringVar(&c.cpuProfile, "cpuprofile", "", "file into which to write CPU profile (if not empty)"+
		" with support for variables (e.g. Timestamp, Pid, Ppid) via Go template"+
		" syntax {{.VarName}}")
	fs.StringVar(&c.memProfile, "memprofile", "", "file into which to write memory profile (if not empty)"+
		" with support for variables (e.g. Timestamp, Pid, Ppid) via Go template"+
		" syntax {{.VarName}}")
	fs.IntVar(&c.reqConcurrency, "req-concurrency", 0, fmt.Sprintf("number of RPC requests to process concurrently,"+
		" defaults to %d, concurrency lower than 2 is not recommended", langserver.DefaultConcurrency()))

	fs.Usage = func() { c.Ui.Error(c.Help()) }

	return fs
}

func (c *ServeCommand) Run(args []string) int {
	f := c.flags()
	if err := f.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s", err))
		return 1
	}

	if c.cpuProfile != "" {
		stop, err := writeCpuProfileInto(c.cpuProfile)
		defer stop()
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	}

	if c.memProfile != "" {
		defer writeMemoryProfileInto(c.memProfile)
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
		ctx = lsctx.WithTerraformExecLogPath(ctx, c.tfExecLogPath)
		logger.Printf("Terraform executions will be logged to %s "+
			"(interpolated at the time of execution)", c.tfExecLogPath)
		logger.Println("[WARN] -tf-log-file is deprecated in favor of `terraformLogFilePath` LSP config option")
	}

	// Setting this option as a CLI flag is deprecated
	// in favor of `terraformExecTimeout` LSP config option.
	// This validation code is duplicated, make changes accordingly.
	if c.tfExecTimeout != "" {
		d, err := time.ParseDuration(c.tfExecTimeout)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to parse Terraform timeout: %s", err))
			return 1
		}
		ctx = lsctx.WithTerraformExecTimeout(ctx, d)
		logger.Printf("Terraform execution timeout set to %s", d)
		logger.Println("[WARN] -tf-exec-timeout is deprecated in favor of `terraformExecTimeout` LSP config option")
	}

	// Setting this option as a CLI flag is deprecated
	// in favor of `terraformExecPath` LSP config option.
	// This validation code is duplicated, make changes accordingly.
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

		ctx = lsctx.WithTerraformExecPath(ctx, path)
		logger.Printf("Terraform exec path set to %q", path)
		logger.Println("[WARN] -tf-exec is deprecated in favor of `terraformExecPath` LSP config option")
	}

	if c.reqConcurrency != 0 {
		ctx = langserver.WithRequestConcurrency(ctx, c.reqConcurrency)
		logger.Printf("Custom request concurrency set to %d", c.reqConcurrency)
	}

	logger.Printf("Starting terraform-ls %s", c.Version)

	ctx = lsctx.WithLanguageServerVersion(ctx, c.Version)

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

type stopFunc func() error

func writeCpuProfileInto(rawPath string) (stopFunc, error) {
	path, err := pathtpl.ParseRawPath("cpuprofile-path", rawPath)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("could not create CPU profile: %s", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		return f.Close, fmt.Errorf("could not start CPU profile: %s", err)
	}

	return func() error {
		pprof.StopCPUProfile()
		return f.Close()
	}, nil
}

func writeMemoryProfileInto(rawPath string) error {
	path, err := pathtpl.ParseRawPath("memprofile-path", rawPath)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create memory profile: %s", err)
	}
	defer f.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("could not write memory profile: %s", err)
	}

	return nil
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
