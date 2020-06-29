package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/sourcegraph/go-lsp"
)

func (h *logHandler) TextDocumentComplete(ctx context.Context, params lsp.CompletionParams) (lsp.CompletionList, error) {
	var list lsp.CompletionList

	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return list, err
	}

	cc, err := lsctx.ClientCapabilities(ctx)
	if err != nil {
		return list, err
	}

	pf, err := lsctx.ParserFinder(ctx)
	if err != nil {
		return list, err
	}

	h.logger.Printf("Finding block at position %#v", params.TextDocumentPositionParams)

	file, err := fs.GetFile(ilsp.FileHandlerFromDocumentURI(params.TextDocument.URI))
	if err != nil {
		return list, err
	}

	hclFile := ihcl.NewFile(file)
	fPos, err := ilsp.FilePositionFromDocumentPosition(params.TextDocumentPositionParams, file)
	if err != nil {
		return list, err
	}

	pos := fPos.Position()

	p, err := pf.ParserForDir(file.Dir())
	if err != nil {
		return list, fmt.Errorf("finding compatible parser failed: %w", err)
	}

	candidates, err := p.CompletionCandidatesAtPos(hclFile, pos)
	if err != nil {
		return list, fmt.Errorf("finding completion items failed: %w", err)
	}

	return ilsp.CompletionList(candidates, pos, cc.TextDocument), nil
}
