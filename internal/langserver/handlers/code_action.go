package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/errors"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func (h *logHandler) TextDocumentCodeAction(ctx context.Context, params lsp.CodeActionParams) []lsp.CodeAction {
	ca, err := h.textDocumentCodeAction(ctx, params)
	if err != nil {
		h.logger.Printf("code action failed: %s", err)
	}

	return ca
}

func (h *logHandler) textDocumentCodeAction(ctx context.Context, params lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	var ca []lsp.CodeAction

	wantedCodeActions := ilsp.SupportedCodeActions.Only(params.Context.Only)
	if len(wantedCodeActions) == 0 {
		return nil, fmt.Errorf("could not find a supported code action to execute for %s, wanted %v",
			params.TextDocument.URI, params.Context.Only)
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return ca, err
	}
	file, err := fs.GetDocument(fh)
	if err != nil {
		return ca, err
	}
	original, err := file.Text()
	if err != nil {
		return ca, err
	}

	for action := range wantedCodeActions {
		switch action {
		case lsp.Source, lsp.SourceFixAll, ilsp.SourceFormatAll, ilsp.SourceFormatAllTerraformLs:
			tfExec, err := module.TerraformExecutorForModule(ctx, fh.Dir())
			if err != nil {
				return ca, errors.EnrichTfExecError(err)
			}

			h.logger.Printf("formatting document via %q", tfExec.GetExecPath())

			edits, err := formatDocument(ctx, tfExec, original, file)
			if err != nil {
				return ca, err
			}

			ca = append(ca, lsp.CodeAction{
				Title: "Format Document",
				Kind:  lsp.SourceFixAll,
				Edit: lsp.WorkspaceEdit{
					Changes: map[string][]lsp.TextEdit{
						string(fh.URI()): edits,
					},
				},
			})
		}
	}

	return ca, nil
}
