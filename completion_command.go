package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchellh/cli"
	fs "github.com/radeksimko/terraform-ls/internal/filesystem"
	"github.com/radeksimko/terraform-ls/internal/terraform/exec"
	"github.com/radeksimko/terraform-ls/internal/terraform/lang"
	lsp "github.com/sourcegraph/go-lsp"
)

type completionCommand struct {
	Ui     cli.Ui
	Logger *log.Logger
}

func (c *completionCommand) Run(args []string) int {
	cmdFlags := defaultFlagSet("completion")

	var offset int
	var atPos string
	cmdFlags.IntVar(&offset, "offset", -1, "byte offset")
	cmdFlags.StringVar(&atPos, "at-pos", "", "at position (line:col)")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	path := cmdFlags.Arg(0)

	// TODO: Allow reading a directory too, iterate over and pre-populate fs.filesystem
	content, err := ioutil.ReadFile(path)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("reading file at %q failed: %s", path, err))
		return 1
	}

	f := fs.NewFile(path, content)

	hclPos := f.ByteOffsetToHCLPos(offset)

	if len(atPos) > 0 {
		parts := strings.Split(atPos, ":")
		line, _ := strconv.Atoi(parts[0])
		col, _ := strconv.Atoi(parts[1])

		lspPos := lsp.Position{Line: line, Character: col}
		hclPos = f.LspPosToHCLPos(lspPos)
		c.Ui.Output(fmt.Sprintf("HCL Position: %#v", hclPos))
	}

	hclBlock, err := f.HclBlockAtPos(hclPos)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("finding config block failed: %s", err))
		return 1
	}

	p := lang.NewParserWithLogger(c.Logger)
	cfgBlock, err := p.ParseBlockFromHcl(hclBlock)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("finding config block failed: %s", err))
		return 1
	}

	wd := filepath.Dir(path)
	tf := exec.NewExecutor(context.Background())
	tf.SetLogger(c.Logger)
	tf.SetWorkdir(wd)
	schemas, err := tf.ProviderSchemas()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("unable to get schemas: %s", err))
		return 1
	}

	err = cfgBlock.LoadSchema(schemas)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("loading schema failed: %s", err))
		return 1
	}

	items, err := cfgBlock.CompletionItemsAtPos(hclPos)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("finding completion items failed: %s", err))
		return 1
	}

	c.Ui.Output(fmt.Sprintf("%#v", items))

	return 0
}

func (c *completionCommand) Help() string {
	helpText := `
Usage: terraform-ls completion [options] [path]

Options:

  -offset Byte offset within the file

`
	return strings.TrimSpace(helpText)
}

func (c *completionCommand) Synopsis() string {
	return "Lists available completion items"
}
