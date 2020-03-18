package commands

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/discovery"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/schema"
	"github.com/hashicorp/terraform-ls/langserver/handlers"
	"github.com/mitchellh/cli"
	lsp "github.com/sourcegraph/go-lsp"
)

type CompletionCommand struct {
	Ui cli.Ui

	atPos string
}

func (c *CompletionCommand) flags() *flag.FlagSet {
	fs := defaultFlagSet("completion")

	fs.StringVar(&c.atPos, "at-pos", "", "at zero-indexed position (line:col)")

	fs.Usage = func() { c.Ui.Error(c.Help()) }

	return fs
}

func (c *CompletionCommand) Run(args []string) int {
	f := c.flags()
	if err := f.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	if f.NArg() != 1 {
		c.Ui.Output(fmt.Sprintf("args is %q", c.flags().Args()))
		return 1
	}

	path := f.Arg(0)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("reading file at %q failed: %s", path, err))
		return 1
	}

	lspUri := lsp.DocumentURI("file://" + path)
	parts := strings.Split(c.atPos, ":")
	if len(parts) != 2 {
		c.Ui.Error(fmt.Sprintf("Error parsing at-pos argument: %q (expected line:col format)\n", c.atPos))
		return 1
	}
	line, err := strconv.Atoi(parts[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing line: %s (expected number)\n", err))
		return 1
	}
	col, err := strconv.Atoi(parts[1])
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing column: %s (expected number)\n", err))
		return 1
	}
	lspPos := lsp.Position{Line: line, Character: col}

	logger := NewLogger(os.Stderr)

	fs := filesystem.NewFilesystem()
	fs.SetLogger(logger)
	fs.Open(lsp.TextDocumentItem{
		URI:     lspUri,
		Text:    string(content),
		Version: 0,
	})

	tfPath, err := discovery.LookPath()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	ss := schema.NewStorage()
	ss.SetLogger(logger)
	ss.SetSynchronous()

	ctx := context.Background()

	dir := fs.URI(lspUri).Dir()

	tf := exec.NewExecutor(ctx, tfPath)
	tf.SetWorkdir(dir)
	version, err := tf.Version()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	err = ss.ObtainSchemasForWorkspace(tf, dir)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	ctx = lsctx.WithFilesystem(fs, ctx)
	ctx = lsctx.WithTerraformVersion(version, ctx)
	ctx = lsctx.WithTerraformExecutor(tf, ctx)
	ctx = lsctx.WithTerraformSchemaReader(ss, ctx)
	ctx = lsctx.WithClientCapabilities(&lsp.ClientCapabilities{}, ctx)

	h := handlers.LogHandler(logger)
	items, err := h.TextDocumentComplete(ctx, lsp.CompletionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: lspUri,
			},
			Position: lspPos,
		},
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(fmt.Sprintf("%#v", items))

	return 0
}

func (c *CompletionCommand) Help() string {
	helpText := `
Usage: terraform-ls completion [options] [path]

` + c.Synopsis() + "\n\n" + helpForFlags(c.flags())
	return strings.TrimSpace(helpText)
}

func (c *CompletionCommand) Synopsis() string {
	return "Lists available completion items"
}
