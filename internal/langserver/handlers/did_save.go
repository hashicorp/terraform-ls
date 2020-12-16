package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/cmd"
	"github.com/hashicorp/terraform-ls/internal/langserver/handlers/command"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (lh *logHandler) TextDocumentDidSave(ctx context.Context, params lsp.DidSaveTextDocumentParams) error {
	optIn, err := lsctx.OptIn(ctx)
	if err != nil {
		return err
	}
	if !optIn.ValidateOnSave {
		return nil
	}

	fh := ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI)
	dh := ilsp.FileHandlerFromDirPath(fh.Dir())

	_, err = command.TerraformValidateHandler(ctx, cmd.CommandArgs{
		"uri": dh.URI(),
	})

	return err
}
