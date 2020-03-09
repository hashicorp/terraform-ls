package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/langserver/handlers"
	"github.com/mitchellh/cli"
	lsp "github.com/sourcegraph/go-lsp"
)

type CompletionCommand struct {
	Ui     cli.Ui
	Logger *log.Logger
}

func (c *CompletionCommand) Run(args []string) int {
	cmdFlags := defaultFlagSet("completion")

	var atPos string
	cmdFlags.StringVar(&atPos, "at-pos", "", "at zero-indexed position (line:col)")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	path := cmdFlags.Arg(0)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("reading file at %q failed: %s", path, err))
		return 1
	}

	lspUri := lsp.DocumentURI("file://" + path)
	parts := strings.Split(atPos, ":")
	if len(parts) != 2 {
		c.Ui.Error(fmt.Sprintf("Error parsing at-pos argument: %q (expected line:col format)\n", atPos))
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

	fs := filesystem.NewFilesystem()
	fs.SetLogger(c.Logger)
	fs.Open(lsp.TextDocumentItem{
		URI:     lspUri,
		Text:    string(content),
		Version: 0,
	})

	ctx := context.Background()
	ctx = lsctx.WithFilesystem(fs, ctx)
	ctx = lsctx.WithTerraformExecutor(exec.NewExecutor(ctx), ctx)
	ctx = lsctx.WithClientCapabilities(&lsp.ClientCapabilities{}, ctx)

	h := handlers.LogHandler(c.Logger)
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
Usage: terraform-ls completion -at-pos=line:col <path>

Options:

  -at-pos at zero-indexed position (line:col)

`
	return strings.TrimSpace(helpText)
}

func (c *CompletionCommand) Synopsis() string {
	return "Lists available completion items"
}
