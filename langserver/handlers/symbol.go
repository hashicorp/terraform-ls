package handlers

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/sourcegraph/go-lsp"
)

func (h *logHandler) TextDocumentSymbol(ctx context.Context, params lsp.DocumentSymbolParams) ([]lsp.SymbolInformation, error) {

	h.logger.Printf("SymbolsTest")

	var symbols []lsp.SymbolInformation

	fs, err := lsctx.DocumentStorage(ctx)
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

	blocks, err := hclFile.Blocks()
	if err != nil {
		return symbols, err
	}

	for _, block := range blocks {
		hclBlock, _ := hclsyntax.ParseBlockFromTokens(block.Tokens())
		labels := hclBlock.Labels
		if len(labels) == 0 || len(labels) > 2 {
			h.logger.Printf("Block with no or more than 2 labels...skipping")
			continue
		}
		var name string
		// this is cheating, did not want to deal with LabelSchema
		if len(labels) == 1 {
			name = fmt.Sprintf("provider.%s", labels[0])
		} else if len(labels) == 2 {
			name = fmt.Sprintf("resourceordatasource.%s.%s", labels[0], labels[1])
		}
		symbols = append(symbols, lsp.SymbolInformation{
			Name: name,
			Kind: lsp.SKStruct,
			Location: lsp.Location{
				URI:   params.TextDocument.URI,
				Range: ilsp.HCLRangeToLSP(hclBlock.LabelRanges[0]),
			},
		})
	}

	h.logger.Printf("SymbolsTest %+v", symbols)

	return symbols, nil

}
