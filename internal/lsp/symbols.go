package lsp

import (
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/zclconf/go-cty/cty"
)

func DocumentSymbols(sbs []decoder.Symbol, caps lsp.DocumentSymbolClientCapabilities) []lsp.DocumentSymbol {
	symbols := make([]lsp.DocumentSymbol, 0)

	for _, s := range sbs {
		symbol, ok := documentSymbol(s, caps)
		if !ok {
			// skip symbol not supported by client
			continue
		}
		symbols = append(symbols, symbol)
	}
	return symbols
}

func documentSymbol(symbol decoder.Symbol, caps lsp.DocumentSymbolClientCapabilities) (lsp.DocumentSymbol, bool) {
	kind, ok := symbolKind(symbol, caps.SymbolKind.ValueSet)
	if !ok {
		return lsp.DocumentSymbol{}, false
	}

	ds := lsp.DocumentSymbol{
		Name:           symbol.Name(),
		Kind:           kind,
		Range:          HCLRangeToLSP(symbol.Range()),
		SelectionRange: HCLRangeToLSP(symbol.Range()),
	}
	if caps.HierarchicalDocumentSymbolSupport {
		ds.Children = DocumentSymbols(symbol.NestedSymbols(), caps)
	}
	return ds, true
}

func symbolKind(symbol decoder.Symbol, supported []lsp.SymbolKind) (lsp.SymbolKind, bool) {
	switch s := symbol.(type) {
	case *decoder.BlockSymbol:
		kind, ok := supportedSymbolKind(supported, lsp.Class)
		if ok {
			return kind, true
		}
	case *decoder.AttributeSymbol:
		kind, ok := exprSymbolKind(s.ExprKind, supported)
		if ok {
			return kind, true
		}
	case *decoder.ExprSymbol:
		kind, ok := exprSymbolKind(s.ExprKind, supported)
		if ok {
			return kind, true
		}
	}

	return lsp.SymbolKind(0), false
}

func exprSymbolKind(symbolKind lang.SymbolExprKind, supported []lsp.SymbolKind) (lsp.SymbolKind, bool) {
	switch k := symbolKind.(type) {
	case lang.LiteralTypeKind:
		switch k.Type {
		case cty.Bool:
			return supportedSymbolKind(supported, lsp.Boolean)
		case cty.String:
			return supportedSymbolKind(supported, lsp.String)
		case cty.Number:
			return supportedSymbolKind(supported, lsp.Number)
		}
	case lang.TraversalExprKind:
		return supportedSymbolKind(supported, lsp.Constant)
	case lang.TupleConsExprKind:
		return supportedSymbolKind(supported, lsp.Array)
	case lang.ObjectConsExprKind:
		return supportedSymbolKind(supported, lsp.Struct)
	}

	return supportedSymbolKind(supported, lsp.Variable)
}

func supportedSymbolKind(supported []lsp.SymbolKind, kind lsp.SymbolKind) (lsp.SymbolKind, bool) {
	for _, s := range supported {
		if s == kind {
			return s, true
		}
	}
	return lsp.SymbolKind(0), false
}
