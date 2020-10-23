package lsp

import (
	"github.com/hashicorp/hcl-lang/decoder"
	lsp "github.com/sourcegraph/go-lsp"
)

func ConvertSymbols(uri lsp.DocumentURI, sbs []decoder.Symbol) []lsp.SymbolInformation {
	symbols := make([]lsp.SymbolInformation, len(sbs))
	for i, s := range sbs {
		symbols[i] = lsp.SymbolInformation{
			Name: s.Name(),
			Kind: lsp.SKClass, // most applicable kind for now
			Location: lsp.Location{
				Range: HCLRangeToLSP(s.Range()),
				URI:   uri,
			},
		}
	}
	return symbols
}
