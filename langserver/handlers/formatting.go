package handlers

import (
	"context"
	"os"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/hcl"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/sourcegraph/go-lsp"
)

func (h *logHandler) TextDocumentFormatting(ctx context.Context, params lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
	var edits []lsp.TextEdit

	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return edits, err
	}

	tf, err := lsctx.TerraformExecutor(ctx)
	if err != nil {
		return edits, err
	}
	// Input is sent to stdin -> no need for a meaningful workdir
	tf.SetWorkdir(os.TempDir())

	fh := ilsp.FileHandler(params.TextDocument.URI)
	file, err := fs.GetFile(fh)
	if err != nil {
		return edits, err
	}

	output, err := tf.Format(file.Text())
	if err != nil {
		return edits, err
	}

	f := hcl.NewFile(file)
	changes, err := f.Diff(output)
	if err != nil {
		return edits, err
	}

	return ilsp.TextEdits(changes), nil
}
