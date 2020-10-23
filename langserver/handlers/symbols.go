package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/sourcegraph/go-lsp"
)

func (h *logHandler) TextDocumentSymbol(ctx context.Context, params lsp.DocumentSymbolParams) ([]lsp.SymbolInformation, error) {
	var symbols []lsp.SymbolInformation

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return symbols, err
	}

	df, err := lsctx.DecoderFinder(ctx)
	if err != nil {
		return symbols, err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return symbols, err
	}

	d, err := df.DecoderForDir(file.Dir())
	if err != nil {
		return symbols, fmt.Errorf("finding compatible decoder failed: %w", err)
	}

	sbs, err := d.SymbolsInFile(file.Filename())
	if err != nil {
		return symbols, err
	}

	return ilsp.ConvertSymbols(params.TextDocument.URI, sbs), nil
}
