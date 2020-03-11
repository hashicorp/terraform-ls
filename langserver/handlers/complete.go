package handlers

import (
	"context"
	"fmt"
	"time"

	lsctx "github.com/hashicorp/terraform-ls/internal/context"
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

	h.logger.Printf("Finding block at position %#v", params.TextDocumentPositionParams)
	hclBlock, hclPos, err := fs.HclBlockAtDocPosition(params.TextDocumentPositionParams)
	if err != nil {
		return list, fmt.Errorf("finding config block failed: %s", err)
	}
	h.logger.Printf("HCL block found at HCL pos %#v", hclPos)

	tf, err := lsctx.TerraformExecutor(ctx)
	if err != nil {
		return list, err
	}

	uri := fs.URI(params.TextDocumentPositionParams.TextDocument.URI)
	wd := uri.Dir()
	tf.SetWorkdir(wd)

	tfVersion, err := tf.Version()
	if err != nil {
		return list, err
	}

	p, err := lang.FindCompatibleParser(tfVersion)
	if err != nil {
		return list, err
	}
	p.SetLogger(h.logger)
	p.SetCapabilities(cc.TextDocument)

	cfgBlock, err := p.ParseBlockFromHCL(hclBlock)
	if err != nil {
		return list, fmt.Errorf("finding config block failed: %s", err)
	}

	h.logger.Printf("Retrieving schemas for %q ...", wd)
	start := time.Now()
	schemas, err := tf.ProviderSchemas()
	if err != nil {
		return list, fmt.Errorf("unable to get schemas: %s", err)
	}
	h.logger.Printf("Schemas retrieved in %s ...", time.Since(start))

	err = cfgBlock.LoadSchema(schemas)
	if err != nil {
		return list, fmt.Errorf("loading schema failed: %s", err)
	}

	list, err = cfgBlock.CompletionItemsAtPos(hclPos)
	if err != nil {
		return list, fmt.Errorf("finding completion items failed: %s", err)
	}

	return list, nil
}
