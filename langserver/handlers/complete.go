package handlers

import (
	"context"
	"fmt"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	"github.com/hashicorp/terraform-ls/internal/hcl"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/hashicorp/terraform-ls/internal/terraform/lang"
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

	sr, err := lsctx.TerraformSchemaReader(ctx)
	if err != nil {
		return list, err
	}

	tfVersion, err := lsctx.TerraformVersion(ctx)
	if err != nil {
		return list, err
	}

	h.logger.Printf("Finding block at position %#v", params.TextDocumentPositionParams)

	file, err := fs.GetFile(ilsp.FileHandler(params.TextDocument.URI))
	if err != nil {
		return list, err
	}
	hclFile := hcl.NewFile(file)
	fPos, err := ilsp.FilePositionFromDocumentPosition(params.TextDocumentPositionParams, file)
	if err != nil {
		return list, err
	}

	hclBlock, hclPos, err := hclFile.BlockAtPosition(fPos)
	if err != nil {
		return list, fmt.Errorf("finding config block failed: %s", err)
	}
	h.logger.Printf("HCL block found at HCL pos %#v", hclPos)

	p, err := lang.FindCompatibleParser(tfVersion)
	if err != nil {
		return list, fmt.Errorf("finding compatible parser failed: %w", err)
	}
	p.SetLogger(h.logger)
	p.SetSchemaReader(sr)

	cfgBlock, err := p.ParseBlockFromHCL(hclBlock)
	if err != nil {
		return list, fmt.Errorf("finding config block failed: %w", err)
	}

	candidates, err := cfgBlock.CompletionCandidatesAtPos(hclPos)
	if err != nil {
		return list, fmt.Errorf("finding completion items failed: %w", err)
	}

	return ilsp.CompletionList(candidates, fPos.Position(), cc.TextDocument), nil
}
