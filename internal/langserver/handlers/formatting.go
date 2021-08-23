package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/filesystem"
	"github.com/hashicorp/terraform-ls/internal/hcl"
	"github.com/hashicorp/terraform-ls/internal/langserver/errors"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func (h *logHandler) TextDocumentFormatting(ctx context.Context, params lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
	var edits []lsp.TextEdit

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return edits, err
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)

	tfExec, err := module.TerraformExecutorForModule(ctx, fh.Dir())
	if err != nil {
		return edits, errors.EnrichTfExecError(err)
	}

	file, err := fs.GetDocument(fh)
	if err != nil {
		return edits, err
	}

	original, err := file.Text()
	if err != nil {
		return edits, err
	}

	h.logger.Printf("formatting document via %q", tfExec.GetExecPath())

	edits, err = formatDocument(ctx, tfExec, original, file)
	if err != nil {
		return edits, err
	}

	return edits, nil
}

func formatDocument(ctx context.Context, tfExec exec.TerraformExecutor, original []byte, file filesystem.Document) ([]lsp.TextEdit, error) {
	var edits []lsp.TextEdit

	formatted, err := tfExec.Format(ctx, original)
	if err != nil {
		return edits, err
	}

	changes := hcl.Diff(file, original, formatted)

	return ilsp.TextEditsFromDocumentChanges(changes), nil
}
