// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"

	"github.com/hashicorp/terraform-ls/internal/algolia"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver"
	"github.com/hashicorp/terraform-ls/internal/langserver/handlers"
	"github.com/hashicorp/terraform-ls/internal/logging"
	"github.com/hashicorp/terraform-ls/internal/pathtpl"
	"github.com/mitchellh/cli"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

type ServeCommand struct {
	Ui      cli.Ui
	Version string

	AlgoliaAppID  string
	AlgoliaAPIKey string

	// flags
	port           int
	logFilePath    string
	cpuProfile     string
	memProfile     string
	reqConcurrency int
}

func (c *ServeCommand) flags() *flag.FlagSet {
	fs := defaultFlagSet("serve")

	fs.IntVar(&c.port, "port", 0, "port number to listen on (turns server into TCP mode)")
	fs.StringVar(&c.logFilePath, "log-file", "", "path to a file to log into with support "+
		"for variables (e.g. timestamp, pid, ppid) via Go template syntax {{varName}}")
	fs.StringVar(&c.cpuProfile, "cpuprofile", "", "file into which to write CPU profile (if not empty)"+
		" with support for variables (e.g. timestamp, pid, ppid) via Go template"+
		" syntax {{varName}}")
	fs.StringVar(&c.memProfile, "memprofile", "", "file into which to write memory profile (if not empty)"+
		" with support for variables (e.g. timestamp, pid, ppid) via Go template"+
		" syntax {{varName}}")
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
		os.Interrupt, syscall.SIGTERM)
	defer cancelFunc()

	if c.reqConcurrency != 0 {
		ctx = langserver.WithRequestConcurrency(ctx, c.reqConcurrency)
		logger.Printf("Custom request concurrency set to %d", c.reqConcurrency)
	}

	logger.Printf("Starting terraform-ls %s", c.Version)

	ctx = lsctx.WithLanguageServerVersion(ctx, c.Version)
	if c.AlgoliaAppID != "" && c.AlgoliaAPIKey != "" {
		ctx = algolia.WithCredentials(ctx, c.AlgoliaAppID, c.AlgoliaAPIKey)
	}

	var err error
	shutdownFunc := func(context.Context) error { return nil }

	// TODO: Currently unused until we decide how/where to export data
	tp := trace.NewNoopTracerProvider()
	otel.SetTracerProvider(tp)

	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to init telemetry: %s", err))
		return 1
	}
	defer func() {
		ctx := context.Background()
		err := shutdownFunc(ctx)
		if err != nil {
			logger.Printf("failed to shutdown telemetry: %s", err)
			return
		}
	}()

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

	err = srv.StartAndWait(os.Stdin, os.Stdout)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to start server: %s", err))
		return 1
	}

	return 0
}

func (c *ServeCommand) otelResourceAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.ServiceName("terraform-ls"),
		semconv.ServiceVersion(c.Version),
		attribute.Int("process.pid", os.Getpid()),
		attribute.Int("runtime.NumCPU", runtime.NumCPU()),
		attribute.Int("port", c.port),
		attribute.Int("reqConcurrency", c.reqConcurrency),
	}
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
