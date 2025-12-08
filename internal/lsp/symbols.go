// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"path/filepath"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/uri"
	"github.com/zclconf/go-cty/cty"
)

// defaultSymbols is the list of symbols that were supported by the initial
// version of the LSP. This list is used as a fallback when the client does
// not provide a list of supported symbols.
var defaultSymbols = []lsp.SymbolKind{
	lsp.File,
	lsp.Module,
	lsp.Namespace,
	lsp.Package,
	lsp.Class,
	lsp.Method,
	lsp.Property,
	lsp.Field,
	lsp.Constructor,
	lsp.Enum,
	lsp.Interface,
	lsp.Function,
	lsp.Variable,
	lsp.Constant,
	lsp.String,
	lsp.Number,
	lsp.Boolean,
	lsp.Array,
}

func WorkspaceSymbols(sbs []decoder.Symbol, caps *lsp.WorkspaceSymbolClientCapabilities) []lsp.SymbolInformation {
	symbols := make([]lsp.SymbolInformation, len(sbs))
	supportedSymbols := defaultSymbols
	if caps != nil && caps.SymbolKind != nil {
		supportedSymbols = caps.SymbolKind.ValueSet
	}

	for i, s := range sbs {
		kind, ok := symbolKind(s, supportedSymbols)
		if !ok {
			// skip symbol not supported by client
			continue
		}

		path := filepath.Join(s.Path().Path, s.Range().Filename)
		symbols[i] = lsp.SymbolInformation{
			Name: s.Name(),
			Kind: kind,
			Location: lsp.Location{
				Range: HCLRangeToLSP(s.Range()),
				URI:   lsp.DocumentURI(uri.FromPath(path)),
			},
		}
	}
	return symbols
}

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
	supportedSymbols := defaultSymbols
	if caps.SymbolKind != nil {
		supportedSymbols = caps.SymbolKind.ValueSet
	}

	kind, ok := symbolKind(symbol, supportedSymbols)
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
	case lang.ReferenceExprKind:
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
