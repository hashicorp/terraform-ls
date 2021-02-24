package handlers

import (
	"context"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/decoder"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func (h *logHandler) WorkspaceSymbol(ctx context.Context, params lsp.WorkspaceSymbolParams) ([]lsp.SymbolInformation, error) {
	var symbols []lsp.SymbolInformation

	mm, err := lsctx.ModuleFinder(ctx)
	if err != nil {
		return symbols, err
	}

	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	modules := mm.ListModules()
	for _, mod := range modules {
		d, err := decoder.DecoderForModule(ctx, mod)
		if err != nil {
			return symbols, err
		}

		modSymbols, err := d.Symbols(params.Query)
		if err != nil {
			continue
		}

		symbols = append(symbols, ilsp.SymbolInformation(mod.Path(), modSymbols,
			cc.Workspace.WorkspaceClientCapabilities.Symbol)...)
	}

	return symbols, nil
}
