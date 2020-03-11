package commands

import (
	"context"
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
	Ui     cli.Ui
	Logger *log.Logger
}

func (c *ServeCommand) Run(args []string) int {
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

	hp := handlers.New()
	srv := langserver.NewLangServer(ctx, hp)
	srv.SetLogger(c.Logger)

	if port != -1 {
		srv.StartTCP(fmt.Sprintf("localhost:%d", port))
		return 0
	}

	srv.StartAndWait(os.Stdin, os.Stdout)

	return 0
}

func (c *ServeCommand) Help() string {
	helpText := `
Usage: terraform-ls serve [options] [path]

`
	return strings.TrimSpace(helpText)
}

func (c *ServeCommand) Synopsis() string {
	return "Starts the Language Server"
}
