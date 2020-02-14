package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mitchellh/cli"
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

	ctx, cancelFunc := signalCtx(context.Background(), c.Logger)
	defer cancelFunc()

	srv := langserver.NewLangServer(c.Logger)

	if port != -1 {
		srv.StartTCP(ctx, fmt.Sprintf("localhost:%d", port))
		return 0
	}

	srv.Start(ctx, os.Stdin, os.Stdout)

	return 0
}

func signalCtx(ctx context.Context, logger *log.Logger) (context.Context, func()) {
	ctx, cancelFunc := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			logger.Printf("%s received, stopping server ...", sig)
			cancelFunc()
		case <-ctx.Done():
		}
	}()

	f := func() {
		signal.Stop(sigChan)
		cancelFunc()
	}

	return ctx, f
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
