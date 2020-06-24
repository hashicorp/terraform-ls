package handlers

import (
	"context"

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

	tff, err := lsctx.TerraformExecutorFinder(ctx)
	if err != nil {
		return edits, err
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)
	file, err := fs.GetFile(fh)
	if err != nil {
		return edits, err
	}

	tf, err := tff.TerraformExecutorForDir(fh.Dir())
	if err != nil {
		// TODO: detect no root module found error
		// -> find OS-wide executor instead
		return edits, err
	}

	// TODO: This should probably be FormatWithContext()
	// so it's cancellable on request cancellation
	formatted, err := tf.Format(file.Text())
	if err != nil {
		return edits, err
	}

	changes := hcl.Diff(file, formatted)

	return ilsp.TextEdits(changes), nil
}
