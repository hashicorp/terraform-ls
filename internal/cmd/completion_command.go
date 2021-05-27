package cmd

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-ls/internal/decoder"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/logging"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
	"github.com/mitchellh/cli"
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
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s", err))
		return 1
	}

	if f.NArg() != 1 {
		c.Ui.Output(fmt.Sprintf("args is %q", c.flags().Args()))
		return 1
	}

	path := f.Arg(0)

	path, err := filepath.Abs(path)
	if err != nil {
		c.Ui.Output(err.Error())
		return 1
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("reading file at %q failed: %s", path, err))
		return 1
	}

	fh := ilsp.FileHandlerFromPath(path)

	parts := strings.Split(c.atPos, ":")
	if len(parts) != 2 {
		c.Ui.Error(fmt.Sprintf("Error parsing at-pos argument: %q (expected line:col format)", c.atPos))
		return 1
	}
	line, err := strconv.Atoi(parts[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing line: %s (expected number)", err))
		return 1
	}
	col, err := strconv.Atoi(parts[1])
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing column: %s (expected number)", err))
		return 1
	}
	lspPos := lsp.Position{Line: float64(line), Character: float64(col)}

	logger := logging.NewLogger(os.Stderr)

	fs := filesystem.NewFilesystem()
	fs.SetLogger(logger)
	fs.CreateAndOpenDocument(fh, "terraform", content)

	file, err := fs.GetDocument(fh)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	fPos, err := ilsp.FilePositionFromDocumentPosition(lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: fh.DocumentURI(),
		},
		Position: lspPos,
	}, file)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	ctx := context.Background()
	ss, err := state.NewStateStore()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	modMgr := module.NewSyncModuleManager(ctx, fs, ss.Modules, ss.ProviderSchemas)

	mod, err := modMgr.AddModule(fh.Dir())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	schema, err := modMgr.SchemaForModule(file.Dir())

	if err != nil {
		c.Ui.Error(fmt.Sprintf("failed to find schema: %s", err.Error()))
		return 1
	}

	d, err := decoder.DecoderForModule(ctx, mod)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("failed to find decoder: %s", err.Error()))
		return 1
	}
	d.SetSchema(schema)

	pos := fPos.Position()

	candidates, err := d.CandidatesAtPos(file.Filename(), pos)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("failed to find candidates: %s", err.Error()))
		return 1
	}

	cc := &lsp.ClientCapabilities{}
	items := ilsp.ToCompletionList(candidates, cc.TextDocument)

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
