package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
	"github.com/sourcegraph/go-lsp"
)

func (h *logHandler) TextDocumentSymbol(ctx context.Context, params lsp.DocumentSymbolParams) ([]lsp.SymbolInformation, error) {
	var symbols []lsp.SymbolInformation

	fs, err := lsctx.DocumentStorage(ctx)
	if err != nil {
		return symbols, err
	}

	pf, err := lsctx.ParserFinder(ctx)
	if err != nil {
		return symbols, err
	}

	file, err := fs.GetDocument(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return symbols, err
	}

	text, err := file.Text()
	if err != nil {
		return symbols, err
	}

	hclFile := ihcl.NewFile(file, text)

	// TODO: block until it's available <-pf.ParserLoadingDone()
	// requires https://github.com/hashicorp/terraform-ls/issues/8
	if err := Waiter(func() (bool, error) {
		return pf.IsSchemaLoaded(file.Dir())
	}).Waitf("parser is not available yet for %s", file.Dir()); err != nil {
		return symbols, err
	}

	p, err := pf.ParserForDir(file.Dir())
	if err != nil {
		return symbols, fmt.Errorf("finding compatible parser failed: %w", err)
	}

	blocks, err := p.Blocks(hclFile)
	if err != nil {
		return symbols, err
	}

	for _, block := range blocks {
		symbols = append(symbols, lsp.SymbolInformation{
			Name: symbolName(block),
			Kind: lsp.SKClass, // most applicable kind for now
			Location: lsp.Location{
				Range: ilsp.HCLRangeToLSP(block.Range()),
				URI:   params.TextDocument.URI,
			},
		})
	}

	return symbols, nil

}

func symbolName(b lang.ConfigBlock) string {
	name := b.BlockType()
	for _, l := range b.Labels() {
		name += "." + l.Value
	}
	return name
}
