package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
	lsp "github.com/sourcegraph/go-lsp"
)

func (h *logHandler) TextDocumentHover(ctx context.Context, params lsp.TextDocumentPositionParams) (lsp.Hover, error) {
	fs, err := lsctx.Filesystem(ctx)
	if err != nil {
		return lsp.Hover{}, err
	}

	sr, err := lsctx.TerraformSchemaReader(ctx)
	if err != nil {
		return lsp.Hover{}, err
	}

	tfVersion, err := lsctx.TerraformVersion(ctx)
	if err != nil {
		return lsp.Hover{}, err
	}

	h.logger.Printf("Finding block at position %#v", params)

	file, err := fs.GetFile(ilsp.FileHandler(params.TextDocument.URI))
	if err != nil {
		return lsp.Hover{}, err
	}
	hclFile := ihcl.NewFile(file)
	fPos, err := ilsp.FilePositionFromDocumentPosition(params, file)
	if err != nil {
		return lsp.Hover{}, err
	}

	pos := fPos.Position()

	p, err := lang.FindCompatibleParser(tfVersion)
	if err != nil {
		return lsp.Hover{}, fmt.Errorf("finding compatible parser failed: %w", err)
	}
	p.SetLogger(h.logger)
	p.SetSchemaReader(sr)

	md, err := p.HoverAtPos(hclFile, pos)
	if err != nil {
		return lsp.Hover{}, fmt.Errorf("hover failed: %w", err)
	}

	markedStrings := []lsp.MarkedString{}
	if md != "" {
		markedStrings = []lsp.MarkedString{lsp.RawMarkedString(md)}
	}

	return lsp.Hover{
		Contents: markedStrings,
		Range:    nil,
	}, nil
}
