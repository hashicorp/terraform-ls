package lsp

import (
	"github.com/hashicorp/hcl-lang/decoder"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/zclconf/go-cty/cty"
)

func DocumentSymbols(sbs []decoder.Symbol, caps lsp.DocumentSymbolClientCapabilities) []lsp.DocumentSymbol {
	symbols := make([]lsp.DocumentSymbol, len(sbs))

	for i, s := range sbs {
		kind, ok := symbolKind(s, caps.SymbolKind.ValueSet)
		if !ok {
			// skip symbol not supported by client
			continue
		}

		symbols[i] = lsp.DocumentSymbol{
			Name:           s.Name(),
			Kind:           kind,
			Range:          HCLRangeToLSP(s.Range()),
			SelectionRange: HCLRangeToLSP(s.Range()),
		}

		if caps.HierarchicalDocumentSymbolSupport {
			symbols[i].Children = DocumentSymbols(s.NestedSymbols(), caps)
		}
	}
	return symbols
}

func symbolKind(symbol decoder.Symbol, supported []lsp.SymbolKind) (lsp.SymbolKind, bool) {
	switch s := symbol.(type) {
	case *decoder.BlockSymbol:
		return supportedSymbolKind(supported, lsp.Class)
	case *decoder.AttributeSymbol:
		// Only primitive types are supported at this point
		switch s.Type {
		case cty.Bool:
			return supportedSymbolKind(supported, lsp.Boolean)
		case cty.String:
			return supportedSymbolKind(supported, lsp.String)
		case cty.Number:
			return supportedSymbolKind(supported, lsp.Number)
		}

		return supportedSymbolKind(supported, lsp.Variable)
	}

	return lsp.SymbolKind(0), false
}

func supportedSymbolKind(supported []lsp.SymbolKind, kind lsp.SymbolKind) (lsp.SymbolKind, bool) {
	for _, s := range supported {
		if s == kind {
			return s, true
		}
	}
	return lsp.SymbolKind(0), false
}
