package handlers

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/hcl"
	"github.com/hashicorp/terraform-ls/internal/langserver/errors"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func (svc *service) TextDocumentFormatting(ctx context.Context, params lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
	var edits []lsp.TextEdit

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)

	tfExec, err := module.TerraformExecutorForModule(ctx, dh.Dir.Path())
	if err != nil {
		return edits, errors.EnrichTfExecError(err)
	}

	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return edits, err
	}

	edits, err = svc.formatDocument(ctx, tfExec, doc.Text, dh)
	if err != nil {
		return edits, err
	}

	return edits, nil
}

func (svc *service) formatDocument(ctx context.Context, tfExec exec.TerraformExecutor, original []byte, dh document.Handle) ([]lsp.TextEdit, error) {
	var edits []lsp.TextEdit

	svc.logger.Printf("formatting document via %q", tfExec.GetExecPath())

	startTime := time.Now()
	formatted, err := tfExec.Format(ctx, original)
	if err != nil {
		svc.logger.Printf("Failed 'terraform fmt' in %s", time.Now().Sub(startTime))
		return edits, err
	}
	svc.logger.Printf("Finished 'terraform fmt' in %s", time.Now().Sub(startTime))

	changes := hcl.Diff(dh, original, formatted)

	return ilsp.TextEditsFromDocumentChanges(changes), nil
}
