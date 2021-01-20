package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func (h *logHandler) TextDocumentSymbol(ctx context.Context, params lsp.DocumentSymbolParams) ([]lsp.SymbolInformation, error) {
	var symbols []lsp.SymbolInformation

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return symbols, err
	}

	mf, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return symbols, err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return symbols, err
	}

	mod, err := mf.ModuleByPath(file.Dir())
	if err != nil {
		return symbols, err
	}

	d, err := module.DecoderForModule(mod)
	if err != nil {
		return symbols, err
	}

	sbs, err := d.SymbolsInFile(file.Filename())
	if err != nil {
		return symbols, err
	}

	return ilsp.ConvertSymbols(params.TextDocument.URI, sbs), nil
}
