package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/langserver/diagnostics"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) TextDocumentDidSave(ctx context.Context, params lsp.DidSaveTextDocumentParams) error {
	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return err
	}

	rmf, err := lsctx.RootModuleFinder(ctx)
	if err != nil {
		return err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return err
	}

	rm, err := rmf.RootModuleByPath(file.Dir())
	if err != nil {
		return err
	}

	wasInit, err := rm.WasInitialized()
	if err != nil {
		h.logger.Printf("error checking if rootmodule was initialized: %s", err)
	}
	if !wasInit {
		return nil
	}

	diags, err := lsctx.Diagnostics(ctx)
	if err != nil {
		return err
	}

	hclDiags, err := rm.ExecuteTerraformValidate(ctx)
	if err != nil {
		return err
	}
	diags.Publish(ctx, rm.Path(), diagnostics.FromHCLMap(hclDiags), "validate")

	return nil
}
