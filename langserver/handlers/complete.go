package handlers

import (
	"context"
	"fmt"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	lsctx "github.com/hashicorp/terraform-ls/internal/context"
	ihcl "github.com/hashicorp/terraform-ls/internal/hcl"
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
	hclFile := ihcl.NewFile(file)
	fPos, err := ilsp.FilePositionFromDocumentPosition(params.TextDocumentPositionParams, file)
	if err != nil {
		return list, err
	}

	pos := fPos.Position()

	p, err := lang.FindCompatibleParser(tfVersion)
	if err != nil {
		return list, fmt.Errorf("finding compatible parser failed: %w", err)
	}
	p.SetLogger(h.logger)
	p.SetSchemaReader(sr)

	tokens, err := hclFile.BlockTokensAtPosition(pos)
	if err != nil {
		if ihcl.IsNoBlockFoundErr(err) {
			return ilsp.CompletionList(p.BlockTypeCandidates(tokens, pos), cc.TextDocument), nil
		}

		return list, fmt.Errorf("finding HCL block failed: %#v", err)
	}

	h.logger.Printf("HCL block found at HCL pos %#v", pos)
	candidates, err := h.completeBlock(p, tokens, pos)
	if err != nil {
		return list, fmt.Errorf("finding completion items failed: %w", err)
	}

	return ilsp.CompletionList(candidates, fPos.Position(), cc.TextDocument), nil
}

func (h *logHandler) completeBlock(p lang.Parser, tokens hclsyntax.Tokens, pos hcl.Pos) (lang.CompletionCandidates, error) {
	cfgBlock, err := p.ParseBlockFromTokens(tokens)
	if err != nil {
		return nil, fmt.Errorf("finding config block failed: %w", err)
	}
	h.logger.Printf("Configuration block %q parsed", cfgBlock.BlockType())

	return cfgBlock.CompletionCandidatesAtPos(pos)
}
