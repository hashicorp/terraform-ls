package handlers

import (
	"context"

	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (svc *service) TextDocumentSymbol(ctx context.Context, params lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	var symbols []lsp.DocumentSymbol

	cc, err := ilsp.ClientCapabilities(ctx)
	if err != nil {
		return symbols, err
	}

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return symbols, err
	}

	d, err := svc.decoderForDocument(ctx, doc)
	if err != nil {
		return symbols, err
	}

	sbs, err := d.SymbolsInFile(doc.Filename)
	if err != nil {
		return symbols, err
	}

	return ilsp.DocumentSymbols(sbs, cc.TextDocument.DocumentSymbol), nil
}
