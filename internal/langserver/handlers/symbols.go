package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) TextDocumentSymbol(ctx context.Context, params lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	var symbols []lsp.DocumentSymbol

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return symbols, err
	}

	cc, err := lsctx.ClientCapabilities(ctx)
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

	d, err := decoderForDocument(ctx, mod, file.LanguageID())
	if err != nil {
		return symbols, err
	}

	sbs, err := d.SymbolsInFile(file.Filename())
	if err != nil {
		return symbols, err
	}

	return ilsp.DocumentSymbols(sbs, cc.TextDocument.DocumentSymbol), nil
}
